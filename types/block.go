package types

import (
	"bytes"
	"encoding/gob"
	"log"
)

// 定义区块结构
type Block struct {
	Index        int64
	Hash         []byte
	PreviousHash []byte
	Timestamp    int64
	Data         []byte
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
