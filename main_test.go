package main

import (
	"testing"
	"encoding/json"
	"github.com/yurencloud/yubtc/types"
	. "github.com/yurencloud/yubtc/types"
)

func TestMains(t *testing.T)  {
	var genesisBlock = Block{
		0,
		[]byte("ba8613cd3c6c6d714cbdd14b8a1c03e59331a96247f9b3d62278e8d97e1531e1"),
		[]byte{},
		1530105476,
		[]byte("genesis block"),
	}
	str, _ := json.Marshal(genesisBlock)
	t.Log(string(str))
	message := types.Message{
		1,
		genesisBlock.Serialize(),
	}
	str2, _ := json.Marshal(message)
	t.Log(string(str2))
	message2 := types.Message{}
	json.Unmarshal([]byte(str2), &message2)
	t.Log(message2)
	t.Log(message2.Type)
	block := *DeserializeBlock(message2.Data)
	t.Log(block.Timestamp)

}