package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	bolt "github.com/boltdb/bolt"
	"io"
	"log"
	"os"
)

const dbFile = "blockchain_%s.db"
const genesisDBFile = "genesis.db"
const blocksBucket = "blocks"
const genesisCoinbaseData = "created by parkjp"

type BlockChain struct {
	tip []byte
	db  *bolt.DB
}

func (blockChain *BlockChain) AddBlock(block *Block) {
	err := blockChain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		blockInDb := b.Get(block.Hash)

		if blockInDb != nil {
			return nil
		}

		blockData := block.Serialize()
		err := b.Put(block.Hash, blockData)
		if err != nil {
			log.Panic(err)
		}

		lastHash := b.Get([]byte("l"))
		lastBlockData := b.Get(lastHash)
		lastBlock := DeserializeBlock(lastBlockData)

		if block.Height > lastBlock.Height {
			err = b.Put([]byte("l"), block.Hash)
			if err != nil {
				log.Panic(err)
			}
			blockChain.tip = block.Hash
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

func CreateBlockchain(address, nodeID string) *BlockChain {
	dbFile := fmt.Sprintf(dbFile, nodeID)
	if dbExists(dbFile) {
		fmt.Println("Blockchain already exists.")
		os.Exit(1)
	}

	var tip []byte

	cbtx := NewCoinbaseTX(address, genesisCoinbaseData)
	genesis := NewGenesisBlock(cbtx)

	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil {
			log.Panic(err)
		}

		err = b.Put(genesis.Hash, genesis.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), genesis.Hash)
		if err != nil {
			log.Panic(err)
		}
		tip = genesis.Hash

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	bc := BlockChain{tip, db}

	return &bc
}

func NewBlockChain(nodeID string) *BlockChain {
	dbFile := fmt.Sprintf(dbFile, nodeID)
	if dbExists(dbFile) == false {
		fmt.Println("Starting Blockchain...")
		genesis, err := os.Open(genesisDBFile)
		if err != nil {
			log.Panic(err)
		}
		defer genesis.Close()

		file, err := os.Create(dbFile)
		if err != nil {
			log.Panic(err)
		}

		_, err = io.Copy(file, genesis)
		if err != nil {
			log.Panic(err)
		}
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		tip = bucket.Get([]byte("l"))

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	blockchain := BlockChain{tip, db}
	return &blockchain
}

func (blockChain *BlockChain) FindUTXO() map[string]TxOutputs {
	UTXO := make(map[string]TxOutputs)
	spentTXOs := make(map[string][]int)
	bci := blockChain.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Vout {
				if spentTXOs[txID] != nil {
					for _, spentOutIdx := range spentTXOs[txID] {
						if spentOutIdx == outIdx {
							continue Outputs
						}
					}
				}

				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}

			if tx.IsCoinbase() == false {
				for _, in := range tx.Vin {
					inTxID := hex.EncodeToString(in.Txid)
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return UTXO
}

func (blockChain *BlockChain) FindTransaction(ID []byte) (Transaction, error) {
	bci := blockChain.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("transaction is not found")
}

func (blockChain *BlockChain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := blockChain.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}

		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTXs)
}

func (blockChain *BlockChain) MineBlock(transactions []*Transaction) *Block {
	var lastHash []byte
	var lastHeight int

	for _, tx := range transactions {
		if blockChain.VerifyTransaction(tx) != true {
			log.Panic("ERROR: Invalid transaction")
		}
	}

	err := blockChain.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		lastHash = bucket.Get([]byte("l"))

		blockData := bucket.Get(lastHash)
		block := DeserializeBlock(blockData)

		lastHeight = block.Height

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	newBlock := NewBlock(transactions, lastHash, lastHeight+1)

	err = blockChain.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		err := bucket.Put(newBlock.Hash, newBlock.Serialize())

		if err != nil {
			log.Panic(err)
		}

		err = bucket.Put([]byte("l"), newBlock.Hash)
		if err != nil {
			log.Panic(err)
		}

		blockChain.tip = newBlock.Hash

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	return newBlock
}

func (blockChain *BlockChain) VerifyTransaction(transaction *Transaction) bool {
	if transaction.IsCoinbase() {
		return true
	}

	prevTXs := make(map[string]Transaction)

	for _, vin := range transaction.Vin {
		prevTX, err := blockChain.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}

		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return transaction.Verify(prevTXs)
}

func (blockChain *BlockChain) GetBlock(blockHash []byte) (Block, error) {
	var block Block

	err := blockChain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		blockData := b.Get(blockHash)

		if blockData == nil {
			return errors.New("block is not found")
		}

		block = *DeserializeBlock(blockData)

		return nil
	})
	if err != nil {
		return block, err
	}

	return block, nil
}

func (blockChain *BlockChain) GetBlockHashes() [][]byte {
	var blocks [][]byte
	bci := blockChain.Iterator()

	for {
		block := bci.Next()

		blocks = append(blocks, block.Hash)

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return blocks
}

func (blockChain *BlockChain) GetBestHeight() int {
	var lastBlock Block

	err := blockChain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash := b.Get([]byte("l"))
		blockData := b.Get(lastHash)
		lastBlock = *DeserializeBlock(blockData)

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return lastBlock.Height
}

func dbExists(dbFile string) bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}
