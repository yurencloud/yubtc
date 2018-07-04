package types

type Message struct{
	Type MESSAGE_TYPE
	Data []byte
}

func (message Message) GetBlockFromData() Block {
	return Block{

	}
}
