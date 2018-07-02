package main

import (
	"log"
	"context"
	"github.com/yurencloud/yubtc/util"
	"time"
	"fmt"
	"crypto/rand"
	mrand "math/rand"
	"io"
	"github.com/davecgh/go-spew/spew"
	golog "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
	multiAddress "github.com/multiformats/go-multiaddr"
	goLogging "github.com/whyrusleeping/go-logging"
	"sync"
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"flag"
	"github.com/boltdb/bolt"
	"bytes"
	"encoding/gob"
)

const dbFile = "db/blockchian.db" //定义数据文件名
const blocksBucket = "blocks"     //区块桶

var genesisBlock = Block{
	0,
	[]byte("ba8613cd3c6c6d714cbdd14b8a1c03e59331a96247f9b3d62278e8d97e1531e1"),
	[]byte{},
	1530105476,
	[]byte("genesis block"),
}

// 区块链数据库
type BlockChain struct {
	lastedBlockHash []byte   //最新一个区块的hash
	db  *bolt.DB //区块链数据库
}

// 区块链数据库迭代器用于迭代区块
type BlockChainIterator struct {
	currentHash []byte   //当前区块的hash
	db          *bolt.DB //区块链数据库
}

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


// 计算区块哈稀
func calculateHash(index int64, previousHash []byte, timestamp int64, data []byte) []byte {
	var buffer bytes.Buffer //Buffer是一个实现了读写方法的可变大小的字节缓冲
	buffer.Write(util.Int64ToBytes(index))
	buffer.Write(previousHash)
	buffer.Write(util.Int64ToBytes(timestamp))
	buffer.Write([]byte(data))
	return util.Sha256(buffer.Bytes())
}

func calculateHashForBlock(block Block) []byte {
	return calculateHash(block.Index, block.PreviousHash, block.Timestamp, block.Data)
}

// 生成下一个区块
func (blockChain *BlockChain)  generateNextBlock(data []byte) Block {
	previousBlock := blockChain.getLatestBlock()
	index := previousBlock.Index + 1
	previousHash := previousBlock.Hash
	timestamp := time.Now().Unix()
	hash := calculateHash(index, previousHash, timestamp, data)
	return Block{index, hash, previousHash, timestamp, data}
}

func (blockChain *BlockChain) getLatestBlock() *Block {
	var block *Block
	err := blockChain.db.View(func(tx *bolt.Tx) error {
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

func (blockChain *BlockChain) getGenesisBlock() *Block {
	var block *Block
	err := blockChain.db.View(func(tx *bolt.Tx) error {
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
func isValidBlockChain(blockChain *BlockChain) bool {
	// 验证创世区块
	if bytes.Equal(calculateHashForBlock(*blockChain.getGenesisBlock()), genesisBlock.Hash) {
		return false
	}
	// 逐一验证区块链上所有区块与前一区块的完整性
	bci := blockChain.Iterator()
	currentBlock := blockChain.getLatestBlock()
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
func (blockChain *BlockChain) replaceBlockChain(newBlockChain *BlockChain) {
	//查询数据库中最后一块的hash
	lastedBlock := blockChain.getLatestBlock()
	if isValidBlockChain(newBlockChain) && newBlockChain.getLatestBlock().Index > lastedBlock.Index {
		log.Println("Received blockchain is valid. Replacing current blockchain with received blockchain")
		blockChain.updateBlockChain(newBlockChain, lastedBlock.Hash)
	} else {
		log.Println("Received blockchain invalid")
	}
}

// 从指定节点的区块链更新到最新的区块链
func (blockChain *BlockChain) updateBlockChain(newBlockChain *BlockChain, lastedBlockHash []byte)  {
	err := newBlockChain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		c := b.Cursor()
		for k, v := c.Seek(lastedBlockHash); k != nil; k, v = c.Next() {
			blockChain.AddBlock(*DeserializeBlock(v))
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}


//区块链数据库添加区块
func (blockChain *BlockChain) AddBlock(block Block) {
	//在挖掘新块之后，我们将其序列化表示保存到数据块中并更新"l"，该密钥现在存储新块的哈希。
	err := blockChain.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		err := bucket.Put(block.Hash, block.Serialize())
		if err != nil {
			log.Panic(err)
		}
		err = bucket.Put([]byte("l"), block.Hash)
		if err != nil {
			log.Panic(err)
		}
		blockChain.lastedBlockHash = block.Hash
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

// 迭代器
func (blockChain *BlockChain) Iterator() *BlockChainIterator {
	blockChainDbIterator := &BlockChainIterator{blockChain.lastedBlockHash, blockChain.db}
	return blockChainDbIterator
}

// 迭代下一区块(其他是上一个区块，一直到创世区块)
func (i *BlockChainIterator) Prev() *Block {
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

func (blockChain *BlockChain)  Print()  {
	bci := blockChain.Iterator()
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
func Close(bc *BlockChain) error {
	return bc.db.Close()
}

// 获取本地区块链或创建一个新区块链
func GetBlockchain() *BlockChain {
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
	blockChain := BlockChain{lastedBlockHash, db}
	return &blockChain
}


// 创建一个全新区块链
func CreateBlockchain() *BlockChain {
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

	blockChain := BlockChain{lastedBlockHash, db}

	return &blockChain
}


func dbExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

/*
节点的一个重要部分是与其他节点共享和同步区块链。以下规则用于保持网络同步。
当一个节点产生一个新块时，它将它广播到网络
当一个节点连接到一个新的对等体时，它将查询最新的块
当一个节点遇到一个索引大于当前已知块的块时，它会将该块添加到当前链中，或者查询完整的块链。
*/

// 与其他节点通信

// 避免数据竞争，用锁
var mutex = &sync.Mutex{}

// 创建P2P节点
// 参数：监听端口 | 是否加密 | 随机种子
// 会生成p2p配对ID，提供给其他端口使用
func makeBasicHost(listenPort int, security bool, randSeed int64) (host.Host, error) {

	var reader io.Reader
	if randSeed == 0 {
		reader = rand.Reader
	} else {
		reader = mrand.New(mrand.NewSource(randSeed))
	}

	// 生成密钥对，用以验证p2p配对ID
	privateKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, reader)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", listenPort)),
		libp2p.Identity(privateKey),
	}

	basicHost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	hostAddress, _ := multiAddress.NewMultiaddr(fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty()))

	address := basicHost.Addrs()[0]
	fullAddress := address.Encapsulate(hostAddress)
	log.Printf("I am %s\n", fullAddress)
	if security {
		log.Printf("Now run \"go run blockchain.go -l %d -d %s -security\" on a different terminal\n", listenPort+1, fullAddress)
	} else {
		log.Printf("Now run \"go run blockchain.go -l %d -d %s\" on a different terminal\n", listenPort+1, fullAddress)
	}

	return basicHost, nil
}

func handleStream(s net.Stream) {
	log.Println("Got a new stream!")
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	go readData(rw)
	go writeData(rw)
	// 数据流会一直打开，直到你关闭他，或者其他端口关闭他
}

// 读取p2p端口发送过来的数据
func readData(rw *bufio.ReadWriter) {
	blockChain := GetBlockchain()
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		if str == "" {
			return
		}
		if str != "\n" {
			chain := BlockChain{}
			log.Println(str)
			if err := json.Unmarshal([]byte(str), &chain); err != nil {
				log.Fatal(err)
			}
			mutex.Lock()
			blockChain.replaceBlockChain(&chain)
			mutex.Unlock()
		}
	}
}

// 广播写数据
func writeData(rw *bufio.ReadWriter) {
	blockChain := GetBlockchain()
	go func() {
		for {
			time.Sleep(5 * time.Second)
			mutex.Lock()
			blockChainBytes, err := json.Marshal(*blockChain)
			if err != nil {
				log.Println(err)
			}
			mutex.Unlock()
			mutex.Lock()
			rw.WriteString(fmt.Sprintf("%s\n", string(blockChainBytes)))
			rw.Flush()
			mutex.Unlock()
		}
	}()

	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		// 删除换行符号，得到要记录到块的数据
		sendData = strings.Replace(sendData, "\n", "", -1)
		// 生成新块
		newBlock := blockChain.generateNextBlock([]byte(sendData))
		// 验证新块
		if isValidNewBlock(newBlock, *blockChain.getLatestBlock()) {
			mutex.Lock()
			blockChain.AddBlock(newBlock)
			mutex.Unlock()
		}
		blockChainBytes, err := json.Marshal(blockChain)
		if err != nil {
			log.Println(err)
		}
		spew.Dump(*blockChain)
		mutex.Lock()
		rw.WriteString(fmt.Sprintf("%s\n", string(blockChainBytes)))
		rw.Flush()
		mutex.Unlock()
	}
}

func main() {
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(goLogging.INFO) // Change to DEBUG for extra info

	// Parse options from the command line
	listenF := flag.Int("l", 0, "wait for incoming connections")
	target := flag.String("d", "", "target peer to dial")
	security := flag.Bool("security", false, "enable security")
	seed := flag.Int64("seed", 0, "set random seed for id generation")
	flag.Parse()

	if *listenF == 0 {
		log.Fatal("Please provide a port to bind on with -l")
	}

	// 创建p2p节点
	p2pHost, err := makeBasicHost(*listenF, *security, *seed)
	if err != nil {
		log.Fatal(err)
	}

	if *target == "" {
		log.Println("listening for connections")
		// Set a stream handler on host A. /p2p/1.0.0 is
		// a user-defined protocol name.
		p2pHost.SetStreamHandler("/p2p/1.0.0", handleStream)

		select {} // hang forever
		/**** This is where the listener code ends ****/
	} else {
		p2pHost.SetStreamHandler("/p2p/1.0.0", handleStream)

		// The following code extracts target's peer ID from the
		// given multiaddress
		ipfsaddr, err := multiAddress.NewMultiaddr(*target)
		if err != nil {
			log.Fatalln(err)
		}

		pid, err := ipfsaddr.ValueForProtocol(multiAddress.P_IPFS)
		if err != nil {
			log.Fatalln(err)
		}

		peerid, err := peer.IDB58Decode(pid)
		if err != nil {
			log.Fatalln(err)
		}

		// Decapsulate the /ipfs/<peerID> part from the target
		// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
		targetPeerAddr, _ := multiAddress.NewMultiaddr(
			fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
		targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)

		// We have a peer ID and a targetAddr so we add it to the peerstore
		// so LibP2P knows how to contact it
		p2pHost.Peerstore().AddAddr(peerid, targetAddr, peerstore.PermanentAddrTTL)

		log.Println("opening stream")
		// make a new stream from host B to host A
		// it should be handled on host A by the handler we set above because
		// we use the same /p2p/1.0.0 protocol
		s, err := p2pHost.NewStream(context.Background(), peerid, "/p2p/1.0.0")
		if err != nil {
			log.Fatalln(err)
		}

		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
		// 创建一个新进程读写数据
		go writeData(rw)
		go readData(rw)

		select {}

	}
}
