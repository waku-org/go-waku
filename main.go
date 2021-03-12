package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol"
	//node "waku/v2/node"
)

func topicName(name string) string {
	return "topic:" + name
}
func JoinSomeTopic(ctx context.Context, ps *pubsub.PubSub, selfID peer.ID, t string) error {
	// join the pubsub topic
	topic, err := ps.Join(topicName(t))
	if err != nil {
		return err
	}

	// and subscribe to it
	sub, err := topic.Subscribe()
	if err != nil {
		return err
	}

	// start reading messages from the subscription in a loop
	go readLoop(sub, ctx)

	go writeLoop(topic, ctx)

	return nil
}

type Test struct {
	Message string
}

func writeLoop(topic *pubsub.Topic, ctx context.Context) {
	m := Test{
		Message: "Hello",
	}
	msgBytes, err := json.Marshal(m)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		time.Sleep(2 * time.Second)
		fmt.Println("Send 'Hello'...")
		topic.Publish(ctx, msgBytes)
	}

}

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

func main() {
	golog.SetAllLoggers(golog.LevelInfo) // Change to INFO for extra info

	hostAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:5555")
	extAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:5555")

	var r io.Reader
	r = rand.Reader
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.ECDSA, -1, r)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	wakuNode, err := node.New(ctx, prvKey, hostAddr, extAddr)

	if err != nil {
		fmt.Print(err)
	}

	ps, err := protocol.NewWakuRelaySub(ctx, wakuNode.Host)
	if err != nil {
		panic(err)
	}

	err = JoinSomeTopic(ctx, ps, wakuNode.Host.ID(), "test")
	if err != nil {
		panic(err)
	}

	// wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Received signal, shutting down...")

	// shut the node down
	if err := wakuNode.Stop(); err != nil {
		panic(err)
	}
}
