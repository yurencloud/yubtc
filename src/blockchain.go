package src

import (
	"log"
	"github.com/yurencloud/yubtc/utils"
	. "github.com/yurencloud/yubtc/types"
	"time"
	"fmt"
	"os"
	"github.com/boltdb/bolt"
	"bytes"
	"github.com/ethereum/go-ethereum/common"
)

const dbFile = "db/blockchian.db" //定义数据文件名
const blocksBucket = "blocks"     //区块桶

var genesisTransaction = Transaction{
	common.Hex2Bytes("e655f6a5f26dc9b4cac6e46f52336428287759cf81ef5ff10854f69d68f43fa3"),
	TXInput{}
}

var genesisBlock = NewGenesisBlock()

//const genesisTransaction = {
//'txIns': [{'signature': '', 'txOutId': '', 'txOutIndex': 0}],
//'txOuts': [{
//'address': '04bfcab8722991ae774db48f934ca79cfb7dd991229153b9f732ba5334aafcd8e7266e47076996b55a14bf9913ee3145ce0cfc1372ada8ada74bd287450313534a',
//'amount': 50
//}],
//'id': 'e655f6a5f26dc9b4cac6e46f52336428287759cf81ef5ff10854f69d68f43fa3'
//};

// 计算区块哈稀
func calculateHash(index int64, previousHash []byte, timestamp int64, data []byte) []byte {
	var buffer bytes.Buffer //Buffer是一个实现了读写方法的可变大小的字节缓冲
	buffer.Write(utils.Int64ToBytes(index))
	buffer.Write(previousHash)
	buffer.Write(utils.Int64ToBytes(timestamp))
	buffer.Write([]byte(data))
	return utils.Sha256(buffer.Bytes())
}

func calculateHashForBlock(block Block) []byte {
	return calculateHash(block.Index, block.PreviousHash, block.Timestamp, block.Data)
}

//创建并返回创世纪Block


// 生成下一个区块
func generateNextBlock(data []byte) Block {
	previousBlock := blockchain.getLatestBlock()
	index := previousBlock.Index + 1
	previousHash := previousBlock.Hash
	timestamp := time.Now().Unix()
	hash := calculateHash(index, previousHash, timestamp, data)
	return Block{index, hash, previousHash, timestamp, data}
}

func getLatestBlock() *Block {
	var block *Block
	err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash := b.Get([]byte("l"))
		block = DeserializeBlock(b.Get(lastHash))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return block
}

func getGenesisBlock() *Block {
	var block *Block
	err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		block = DeserializeBlock(b.Get([]byte("ba8613cd3c6c6d714cbdd14b8a1c03e59331a96247f9b3d62278e8d97e1531e1")))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return block
}

// 验证新区块的完整性
func isValidNewBlock(newBlock Block, previousBlock Block) bool {
	if previousBlock.Index+1 != newBlock.Index {
		log.Println("invalid block Index")
		return false
	} else if !bytes.Equal(previousBlock.Hash, newBlock.PreviousHash) {
		log.Println("invalid previous Hash")
		return false
	} else if !bytes.Equal(calculateHashForBlock(newBlock), newBlock.Hash) {
		log.Println("invalid Hash")
		return false
	}
	return true
}

// 验证区块链
func isValidBlockchain(blockchain *Blockchain) bool {
	// 验证创世区块
	if bytes.Equal(calculateHashForBlock(*blockchain.getGenesisBlock()), genesisBlock.Hash) {
		return false
	}
	// 逐一验证区块链上所有区块与前一区块的完整性
	bci := blockchain.Iterator()
	currentBlock := blockchain.getLatestBlock()
	for {
		previousBlock := bci.Prev()
		if !isValidNewBlock(*currentBlock, *previousBlock) {
			return false
		}
		currentBlock = previousBlock
	}
	return true
}

// 多节点存在时，各节点区块链长度可能不一致，同步时取最长链
func replaceBlockchain(newBlockchain *Blockchain) {
	//查询数据库中最后一块的hash
	lastedBlock := blockchain.getLatestBlock()
	if isValidBlockchain(newBlockchain) && newBlockchain.getLatestBlock().Index > lastedBlock.Index {
		log.Println("Received blockchain is valid. Replacing current blockchain with received blockchain")
		blockchain.updateBlockchain(newBlockchain, lastedBlock.Hash)
	} else {
		log.Println("Received blockchain invalid")
	}
}

// 从指定节点的区块链更新到最新的区块链
func updateBlockchain(newBlockchain *Blockchain, lastedBlockHash []byte)  {
	err := newBlockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		c := b.Cursor()
		for k, v := c.Seek(lastedBlockHash); k != nil; k, v = c.Next() {
			blockchain.AddBlock(*DeserializeBlock(v))
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}


//区块链数据库添加区块
func AddBlock(block Block) {
	//在挖掘新块之后，我们将其序列化表示保存到数据块中并更新"l"，该密钥现在存储新块的哈希。
	err := blockchain.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		err := bucket.Put(block.Hash, block.Serialize())
		if err != nil {
			log.Panic(err)
		}
		err = bucket.Put([]byte("l"), block.Hash)
		if err != nil {
			log.Panic(err)
		}
		blockchain.lastedBlockHash = block.Hash
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

// 迭代器
func Iterator() *BlockchainIterator {
	blockchainDbIterator := &BlockchainIterator{blockchain.lastedBlockHash, blockchain.db}
	return blockchainDbIterator
}

// 迭代下一区块(其他是上一个区块，一直到创世区块)
func Prev() *Block {
	var block *Block
	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block = DeserializeBlock(encodedBlock)
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	i.currentHash = block.PreviousHash
	return block
}

func Print()  {
	bci := blockchain.Iterator()
	for {
		block := bci.Prev()
		fmt.Printf("============ Block %x ============\n", block.Hash)
		fmt.Printf("Prev. block: %x\n", block.PreviousHash)
		fmt.Printf("\n\n")
		if len(block.PreviousHash) == 0 {
			break
		}
	}
}


// 关闭方法
func Close(bc *Blockchain) error {
	return bc.db.Close()
}

// 获取本地区块链或创建一个新区块链
func GetBlockchain() *Blockchain {
	if dbExists() == false {
		fmt.Println("No existing blockchain found. Now Create One")
		return CreateBlockchain()
	}
	var lastedBlockHash []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastedBlockHash = b.Get([]byte("l"))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	blockchain := Blockchain{lastedBlockHash, db}
	return &blockchain
}


// 创建一个全新区块链
func CreateBlockchain() *Blockchain {
	if dbExists() {
		fmt.Println("Blockchain already exists.")
		os.Exit(1)
	}

	var lastedBlockHash []byte
	genesis := genesisBlock

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
		lastedBlockHash = genesis.Hash

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	blockchain := Blockchain{lastedBlockHash, db}

	return &blockchain
}


func dbExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}


