package main

import (
	"fmt"
	"bufio"
	"time"
	"sync"
	"crypto/rand"
	mrand "math/rand"
	"io"
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
	//blockchain := GetBlockchain()
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		if str == "" {
			return
		}
		if str != "\n" {
			//block := Block{}
			//log.Println(str)
			//if err := json.Unmarshal([]byte(str), &block); err != nil {
			//	log.Fatal(err)
			//}
			mutex.Lock()
			//blockchain.AddBlock(block)
			mutex.Unlock()
		}
	}
}

// 通过readData和writeData得到一个自运行的p2p服务
// 1、 write(ws, queryChainLengthMsg()); // 获取最新的区块长度，收到请求的其他区块会广播自己最新的区块
// 2、          case MessageType.QUERY_LATEST:
//                    write(ws, responseLatestMsg()); 收到最新区块的响应
//3、       case MessageType.RESPONSE_BLOCKCHAIN:
//                    const receivedBlocks = JSONToObject(message.data);
//                    if (receivedBlocks === null) {
//                        console.log('invalid blocks received: %s', JSON.stringify(message.data));
//                        break;
//                    }
//                    handleBlockchainResponse(receivedBlocks);
//                    break; // 处理接收到的最新区块，若刚好接上，就添加到区块链，区块很老就叫他更新，若区块很高，就向他请示全部区块
// 以便自己更新

// 首次会发出查询最新区块的请求
// 之后会定时广播，查询未写入到块的交易池，并更新
// 而如果自己或者其他节点添加了区块，他会主动广播，说我加了新区块，让大家更新
// 在js里，广播是 将websocks保存到数组中，全部遍历发送一遍消息

// 我的解决方案，每1秒对外广播1次自己的最新交易池和最新hash(这个不用查询数据库就可以获得)。
// 接收方更新自己的交易池，同时检查自己的最新区块hash和接收到的最新区块hash是否相同，如果不相同，再检查自己的区块链数据库中有没有这个hash，
// 如果没有，就说明自己落后了，要广播，告诉大家他的最高区块，大家根据他的最高区块，返回50个以内的区块给他，他接收到后添加到自己的区块链中
// 如果他落后大于50，但经过多次广播后，也会同步到最新节点
// 一个区块中最多20笔交易

// 如何通过p2p向自己请求：自己对外writeData，自己也会收到请求，这时自己响应就可以

// js案例中，需要额外的挖矿行为，所以有待处理的交易池。而go案例中，当发生交易时，立即自动进行挖矿行为，立即处理交易并写入块，
// 所以没有交易池。

// 广播写数据
func writeData(rw *bufio.ReadWriter) {
	//blockchain := GetBlockchain()
	// 在一个新进程里循环广播
	go func() {
		for {
			// 每5秒广播最新的区块
			time.Sleep(1 * time.Second)
			mutex.Lock()
			//lastedBlock := blockchain.getLatestBlock()
			//lastedBlockBytes, err := json.Marshal(*lastedBlock)
			//if err != nil {
			//	log.Println(err)
			//}
			mutex.Unlock()
			mutex.Lock()
			// 广播最新区块
			rw.WriteString(fmt.Sprintf("%s\n", string("hello")))
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