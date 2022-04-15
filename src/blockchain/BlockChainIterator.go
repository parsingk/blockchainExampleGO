package blockchain

import "github.com/boltdb/bolt"

type BlockChainIterator struct {
	currentHash []byte
	db          *bolt.DB
}

func (blockChain *BlockChain) Iterator() *BlockChainIterator {
	bci := &BlockChainIterator{blockChain.tip, blockChain.db}
	return bci
}

func (i *BlockChainIterator) Next() *Block {
	var block *Block

	err := i.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		encodedBlock := bucket.Get(i.currentHash)
		block = DeserializeBlock(encodedBlock)
		return nil
	})

	if err != nil {
		//
	}

	i.currentHash = block.PrevBlockHash
	return block
}
