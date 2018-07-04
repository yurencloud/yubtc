package main

import (
	"fmt"
	"encoding/json"
	"bufio"
	"time"
	"sync"
	"strings"
	"crypto/rand"
	"os"
	mrand "math/rand"
	"io"
	"github.com/davecgh/go-spew/spew"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-net"
	multiAddress "github.com/multiformats/go-multiaddr"
	. "github.com/yurencloud/yubtc/types"
	"context"
	"log"
)



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
		log.Printf("Now run \"go run blockchain.go -l %d -d %s -s\" on a different terminal\n", listenPort+1, fullAddress)
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

func handleMessage(message Message)  {
	switch message.Type {
		case QUERY_LATEST:

	}
}

// 读取p2p端口发送过来的数据
func readData(rw *bufio.ReadWriter) {
	blockchain := GetBlockchain()
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		if str == "" {
			return
		}
		if str != "\n" {
			block := Block{}
			log.Println(str)
			if err := json.Unmarshal([]byte(str), &block); err != nil {
				log.Fatal(err)
			}
			mutex.Lock()
			blockchain.AddBlock(block)
			mutex.Unlock()
		}
	}
}

// 广播写数据
func writeData(rw *bufio.ReadWriter) {
	blockchain := GetBlockchain()
	go func() {
		for {
			// 每5秒广播最新的区块
			time.Sleep(1 * time.Second)
			mutex.Lock()
			lastedBlock := blockchain.getLatestBlock()
			lastedBlockBytes, err := json.Marshal(*lastedBlock)
			if err != nil {
				log.Println(err)
			}
			mutex.Unlock()
			mutex.Lock()
			// 广播最新区块
			rw.WriteString(fmt.Sprintf("%s\n", string(lastedBlockBytes)))
			rw.Flush()
			mutex.Unlock()
		}
	}()

	//for {
	//	// 获取用户的输入，若没有，则跳过
	//	fmt.Print("> ")
	//	sendData, err := stdReader.ReadString('\n')
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	// 删除换行符号，得到要记录到块的数据
	//	sendData = strings.Replace(sendData, "\n", "", -1)
	//	// 生成新块
	//	newBlock := blockchain.generateNextBlock([]byte(sendData))
	//	// 验证新块
	//	if isValidNewBlock(newBlock, *blockchain.getLatestBlock()) {
	//		mutex.Lock()
	//		blockchain.AddBlock(newBlock)
	//		mutex.Unlock()
	//	}
	//	newBlockBytes, err := json.Marshal(newBlock)
	//	if err != nil {
	//		log.Println(err)
	//	}
	//	// 打印最新区块
	//	spew.Dump(newBlock)
	//	mutex.Lock()
	//	// 广播最新区块
	//	rw.WriteString(fmt.Sprintf("%s\n", string(newBlockBytes)))
	//	rw.Flush()
	//	mutex.Unlock()
	//}
}