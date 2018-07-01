package main

import (
	"log"
	"strconv"
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
)

// 定义区块结构
type Block struct {
	index        int64
	hash         string
	previousHash string
	timestamp    int64
	data         string
}

var BlockChain = []Block{}

// 创世区块 block
var genesisBlock = Block{
	0,
	"ba8613cd3c6c6d714cbdd14b8a1c03e59331a96247f9b3d62278e8d97e1531e1",
	"",
	1530105476,
	"genesis block",
}

func init()  {
	BlockChain = append(BlockChain, genesisBlock)
}

// 计算区块哈稀
func calculateHash(index int64, previousHash string, timestamp int64, data string) string {
	blockStr := strconv.FormatInt(index, 10) + previousHash + strconv.FormatInt(timestamp, 10) + data
	return util.Sha256(blockStr)
}

func calculateHashForBlock(block Block) string {
	return calculateHash(block.index, block.previousHash, block.timestamp, block.data)
}

// 生成下一个区块
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

// 验证新区块的完整性
func isValidNewBlock(newBlock Block, previousBlock Block) bool {
	if previousBlock.index + 1 != newBlock.index {
		log.Println("invalid block index")
		return false
	} else if previousBlock.hash != newBlock.previousHash {
		log.Println("invalid previous hash")
		return false
	} else if calculateHashForBlock(newBlock) != newBlock.hash {
		log.Println("invalid hash")
		return false
	}

	return true
}

// 验证区块链
func isValidBlockChain(blockChain []Block) bool {
	// 验证创建区块
	if blockChain[0] != genesisBlock {
		return false
	}

	// 逐一验证区块链上所有区块与前一区块的完整性
	for i := 1; i < len(blockChain); i++ {
		if !isValidNewBlock(blockChain[i], blockChain[i-1]) {
			return false
		}
	}

	return true
}

// 多节点存在时，各节点区块链长度可能不一致，同步时取最长链
func replaceBlockChain(newBlockChain []Block)  {
	if isValidBlockChain(newBlockChain) && len(newBlockChain) > len(BlockChain) {
		log.Println("Received blockchain is valid. Replacing current blockchain with received blockchain")
		BlockChain = newBlockChain
	} else {
		log.Println("Received blockchain invalid")
	}
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
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		if str == "" {
			return
		}
		if str != "\n" {
			chain := make([]Block, 0)
			log.Println(str)
			if err := json.Unmarshal([]byte(str), &chain); err != nil {
				log.Fatal(err)
			}
			mutex.Lock()
			if len(chain) > len(BlockChain) {
				BlockChain = chain
				bytes, err := json.MarshalIndent(BlockChain, "", "  ")
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("\x1b[32m%s\x1b[0m> ", string(bytes))
			}
			mutex.Unlock()
		}
	}
}

// 广播写数据
func writeData(rw *bufio.ReadWriter) {
	go func() {
		for {
			time.Sleep(5 * time.Second)
			mutex.Lock()
			// TODO:: 这里内存里的区块链为空，待解决
			bytes, err := json.Marshal(BlockChain)
			if err != nil {
				log.Println(err)
			}
			mutex.Unlock()

			mutex.Lock()
			rw.WriteString(fmt.Sprintf("%s\n", string(bytes)))
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
		newBlock := generateNextBlock(sendData)
		// 验证新块
		if isValidNewBlock(newBlock, BlockChain[len(BlockChain)-1]) {
			mutex.Lock()
			BlockChain = append(BlockChain, newBlock)
			mutex.Unlock()
		}
		bytes, err := json.Marshal(BlockChain)
		if err != nil {
			log.Println(err)
		}
		spew.Dump(BlockChain)
		mutex.Lock()
		rw.WriteString(fmt.Sprintf("%s\n", string(bytes)))
		rw.Flush()
		mutex.Unlock()
	}
}

func main() {
	BlockChain = append(BlockChain, genesisBlock)

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