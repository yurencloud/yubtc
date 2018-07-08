package src

import (
	"bytes"
	"encoding/gob"
	"log"
	"github.com/yurencloud/yubtc/utils"
	"time"
)

// 定义区块结构
type Block struct {
	Index        int64
	Hash         []byte
	PreviousHash []byte
	Timestamp    int64
	Transactions  []*Transaction
	Nonce        int64
}

// 序列化Block
func (b *Block) Serialize() []byte  {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}
	return result.Bytes()
}

//反序列化
func DeserializeBlock(d []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}
	return &block
}

//返回块状事务的hash
func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte
	var txHash []byte
	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID)
	}
	txHash = utils.Sha256(bytes.Join(txHashes, []byte{}))
	return txHash[:]
}


//生成一个新的区块方法
func NewBlock(transactions []*Transaction, prevBlockHash []byte) *Block{
	//GO语言给Block赋值{}里面属性顺序可以打乱，但必须制定元素 如{Timestamp:time.Now().Unix()...}
	block := &Block{Timestamp:time.Now().Unix(), Transactions:transactions, PreviousHash:prevBlockHash, Hash:[]byte{},Nonce:0}

	//工作证明
	pow :=NewProofOfWork(block)
	//工作量证明返回计数器和hash
	nonce, hash := pow.Run()
	block.Hash = hash[:]
	block.Nonce = nonce
	return block
}


//区块校验
func (i *Block) Validate() bool {
	return NewProofOfWork(i).Validate()
}

//创建并返回创世纪Block
func  NewGenesisBlock(coinbase *Transaction) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{})
}

