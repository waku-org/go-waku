package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	golog "github.com/ipfs/go-log/v2"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol"
	ethNodeCrypto "github.com/status-im/status-go/eth-node/crypto"
)

/*
func readLoop(sub *pubsub.Subscription, ctx context.Context) {
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			fmt.Println(err)
			return
		}

		cm := new(Test)
		err = json.Unmarshal(msg.Data, cm)
		if err != nil {
			return
		}

		fmt.Println("Received: " + cm.Message)
	}
}
*/

func main() {
	golog.SetAllLoggers(golog.LevelInfo) // Change to INFO for extra info

	hostAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:5555")
	extAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:5555")

	key := "9ceff459635becbab13190132172fc9612357696c176a9e2b6e22f28a73a54ce"
	prvKey, err := ethNodeCrypto.HexToECDSA(key)

	ctx := context.Background()

	wakuNode, err := node.New(ctx, prvKey, hostAddr, extAddr)
	if err != nil {
		fmt.Print(err)
	}

	wakuNode.MountRelay()

	sub, err := wakuNode.Subscribe(nil)
	if err != nil {
		fmt.Println("Could not subscribe:", err)
	}

	go func(sub chan *protocol.WakuMessage) {
		for {
			fmt.Println("Waiting for a message...")
			x := <-sub
			fmt.Println("Received a message: ", string(x.Payload))
		}
	}(sub)

	for {
		time.Sleep(4 * time.Second)
		fmt.Println("Sending 'Hello World'...")

		var contentTopic uint32 = 1
		var version uint32 = 0

		msg := &protocol.WakuMessage{Payload: []byte("Hello World"), Version: &version, ContentTopic: &contentTopic}
		err = wakuNode.Publish(msg, nil)
		if err != nil {
			fmt.Println("ERROR SENDING MESSAGE", err)
		} else {
			fmt.Println("Sent...")
		}
	}

	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Received signal, shutting down...")

	// shut the node down
	if err := wakuNode.Stop(); err != nil {
		panic(err)
	}
}
