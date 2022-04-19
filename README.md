
# Blockchain Example Code (Golang)

  
```
 $ go mod vendor
 $ go build -mod vendor
 $ go build blockchainTest
 
 $ ./blockchainTest createwallet
 $ ./blockchainTest startnode                                   // normal mode
 $ ./blockchainTest startnode -miner <your wallet address>     // miner mode
```


## Command Line Usage
```
 createblockchain -address <ADDRESS>   
 - Create a blockchain and send genesis block reward to ADDRESS
 
 createwallet 
 - Generates a new key-pair and saves it into the wallet file
 
 getbalance -address <ADDRESS>
 - Get balance of ADDRESS"
 
 listaddresses 
 - Lists all addresses from the wallet file
 
 printchain 
 - Print all the blocks of the blockchain
 
 reindexutxo 
 - Rebuilds the UTXO set
 
 send -from <FROM> -to <TO> -amount <AMOUNT>
 - Send AMOUNT of coins from FROM address to TO.
 
 send -from <FROM> -to <TO> -amount <AMOUNT> -mine 
 - Send AMOUNT of coins from FROM address to TO. Mine on the same node, when -mine is set.

 startnode
 - Start a node with ID specified in NODE_ID
  
 startnode -miner <ADDRESS> 
 - Start a node with ID specified in NODE_ID. -miner enables mining

```
