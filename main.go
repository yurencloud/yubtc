package main

import (
	"github.com/libp2p/go-libp2p-peer"
	"flag"
	"fmt"
	"github.com/libp2p/go-libp2p-peerstore"
	"bufio"
	golog "github.com/ipfs/go-log"
	multiAddress "github.com/multiformats/go-multiaddr"
	goLogging "github.com/whyrusleeping/go-logging"
	"log"
	"context"
	"github.com/gorilla/mux"
)

func main() {
	// 可以改成debug模式查看更多p2p信息
	golog.SetAllLoggers(goLogging.INFO)

	listenF := flag.Int("l", 0, "wait for incoming connections")
	target := flag.String("d", "", "target peer to dial")
	security := flag.Bool("s", false, "enable security")
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
		// 如果没有p2p目标节点，则直接开启一个独立p2p节点
		p2pHost.SetStreamHandler("/p2p/1.0.0", handleStream)
		select {}
	} else {
		// 如果有p2p目标节点
		p2pHost.SetStreamHandler("/p2p/1.0.0", handleStream)

		// 从target中获取要对外暴露的p2p节点地址
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


		targetPeerAddr, _ := multiAddress.NewMultiaddr(fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
		targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)

		// 我们有了peer ID 和 目标地址，所以我们可以将其添加到peerstore，这样LibP2P就知道如何连接他
		p2pHost.Peerstore().AddAddr(peerid, targetAddr, peerstore.PermanentAddrTTL)

		log.Println("opening stream")

		// 新建一个A - B节点的p2p连接
		// A和B都要设置相同的： /p2p/1.0.0 protocol
		s, err := p2pHost.NewStream(context.Background(), peerid, "/p2p/1.0.0")
		if err != nil {
			log.Fatalln(err)
		}

		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
		// 创建一个新进程读写数据
		go writeData(rw)
		go readData(rw)

		// 这个可以主动向其他的p2p发信息
		//// doEcho reads a line of data a stream and writes it back
		//func doEcho(s net.Stream) error {
		//	buf := bufio.NewReader(s)
		//	str, err := buf.ReadString('\n')
		//	if err != nil {
		//	return err
		//}
		//
		//	log.Printf("read: %s\n", str)
		//	_, err = s.Write([]byte(str))
		//	return err
		//}

		select {}

		router := mux.NewRouter()
		InitRouter(router)
	}
}
