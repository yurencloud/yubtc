package main

import (
	"log"
	"strconv"
	"github.com/yurencloud/yubtc/util"
	"time"
)

// 定义block结构
type Block struct {
	index        int64
	hash         string
	previousHash string
	timestamp    int64
	data         string
}

var BlockChain = []Block{}

func init()  {
	// 创世块 block
	genesisBlock := Block{
		0,
		"ba8613cd3c6c6d714cbdd14b8a1c03e59331a96247f9b3d62278e8d97e1531e1",
		"",
		1530105476,
		"genesis block",
	}

	BlockChain = append(BlockChain, genesisBlock)
}

// 计算block哈稀
func calculateHash(index int64, previousHash string, timestamp int64, data string) string {
	blockStr := strconv.FormatInt(index, 10) + previousHash + strconv.FormatInt(timestamp, 10) + data
	return util.Sha256(blockStr)
}

// 生成下一个block
func generateNextBlock(data string) Block {
	previousBlock := getLatestBlock()
	index := previousBlock.index + 1
	previousHash := previousBlock.hash
	timestamp := time.Now().Unix()
	hash := calculateHash(index,previousHash,timestamp,data)
	return Block{index, hash, previousHash, timestamp, data}
}

func getLatestBlock() Block {
	return BlockChain[len(BlockChain)-1]
}

func main() {
	block1 := generateNextBlock("1")
	BlockChain = append(BlockChain, block1)
	log.Print(BlockChain)

	block2 := generateNextBlock("2")
	BlockChain = append(BlockChain, block2)
	log.Print(BlockChain)

	block3 := generateNextBlock("3")
	BlockChain = append(BlockChain, block3)
	log.Print(BlockChain)
}
