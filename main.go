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

func main() {
	golog.SetAllLoggers(golog.LevelInfo) // Change to INFO for extra info

	hostAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:60001")
	extAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:60001")

	key := "9ceff459635becbab13190132172fc9612357696c176a9e2b6e22f28a73a54de"
	prvKey, err := ethNodeCrypto.HexToECDSA(key)

	ctx := context.Background()

	wakuNode, err := node.New(ctx, prvKey, hostAddr, extAddr)
	if err != nil {
		fmt.Print(err)
	}

	wakuNode.MountRelay()
	wakuNode.MountStore()

	sub, err := wakuNode.Subscribe(nil)
	if err != nil {
		fmt.Println("Could not subscribe:", err)
	}

	// Read loop
	go func() {
		for value := range sub.C {
			payload, err := node.DecodePayload(value, &node.KeyInfo{Kind: node.None})
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Println("Received message:", string(payload))
			// sub.Unsubscribe()
		}

	}()

	// Write loop
	go func() {
		for {
			time.Sleep(1 * time.Second)

			var contentTopic uint32 = 1
			var version uint32 = 0

			payload, err := node.Encode([]byte("Hello World"), &node.KeyInfo{Kind: node.None}, 0)
			msg := &protocol.WakuMessage{Payload: payload, Version: &version, ContentTopic: &contentTopic}
			err = wakuNode.Publish(msg, nil)
			if err != nil {
				fmt.Println("Error sending a message", err)
			} else {
				fmt.Println("Sent  message...")
			}
		}
	}()

	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\n\n\nReceived signal, shutting down...")

	// shut the node down
	if err := wakuNode.Stop(); err != nil {
		panic(err)
	}
}
