package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	logging "github.com/ipfs/go-log"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
)

var log = logging.Logger("filter2")

var pubSubTopic = relay.DefaultWakuTopic

const contentTopic = "test"

func main() {
	lvl, err := logging.LevelFromString("info")
	if err != nil {
		panic(err)
	}
	logging.SetAllLoggers(lvl)

	hostAddr1, _ := net.ResolveTCPAddr("tcp", fmt.Sprint("0.0.0.0:60000"))
	hostAddr2, _ := net.ResolveTCPAddr("tcp", fmt.Sprint("0.0.0.0:60001"))

	key1, err := randomHex(32)
	if err != nil {
		log.Error("Could not generate random key")
		return
	}
	prvKey1, err := crypto.HexToECDSA(key1)

	key2, err := randomHex(32)
	if err != nil {
		log.Error("Could not generate random key")
		return
	}
	prvKey2, err := crypto.HexToECDSA(key2)

	ctx := context.Background()

	fullNode, err := node.New(ctx,
		node.WithPrivateKey(prvKey1),
		node.WithHostAddress([]net.Addr{hostAddr1}),
		node.WithWakuRelay(),
		node.WithWakuFilter(),
	)

	err = fullNode.Start()
	if err != nil {
		panic(err)
	}

	lightNode, err := node.New(ctx,
		node.WithPrivateKey(prvKey2),
		node.WithHostAddress([]net.Addr{hostAddr2}),
		node.WithWakuFilter(),
	)

	_, err = lightNode.AddPeer(fullNode.ListenAddresses()[0], filter.FilterID_v20beta1)
	if err != nil {
		log.Info("Error adding filter peer on light node ", err)
	}

	err = lightNode.Start()
	if err != nil {
		panic(err)
	}

	//
	// Setup filter
	//

	// Send FilterRequest from light node to full node
	filterRequest := pb.FilterRequest{
		ContentFilters: []*pb.FilterRequest_ContentFilter{{ContentTopic: contentTopic}},
		Topic:          string(pubSubTopic),
		Subscribe:      true,
	}

	filterChan := make(filter.ContentFilterChan)

	go func() {
		for env := range filterChan {
			log.Info("Light node received msg, ", string(env.Message().Payload))
		}
	}()
	lightNode.SubscribeFilter(ctx, filterRequest, filterChan)

	go writeLoop(ctx, fullNode)
	go readLoop(fullNode)

	go func() {
		// Unsubscribe filter after 5 seconds
		time.Sleep(5 * time.Second)
		filterRequest.Subscribe = false
		lightNode.UnsubscribeFilter(ctx, filterRequest)
	}()
	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\n\n\nReceived signal, shutting down...")

	// shut the nodes down
	fullNode.Stop()
	lightNode.Stop()

}

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func write(ctx context.Context, wakuNode *node.WakuNode, msgContent string) {
	var version uint32 = 0
	var timestamp float64 = float64(time.Now().UnixNano())

	p := new(node.Payload)
	p.Data = []byte(wakuNode.ID() + ": " + msgContent)
	p.Key = &node.KeyInfo{Kind: node.None}

	payload, err := p.Encode(version)

	msg := &pb.WakuMessage{
		Payload:      payload,
		Version:      version,
		ContentTopic: contentTopic,
		Timestamp:    timestamp,
	}

	_, err = wakuNode.Publish(ctx, msg, nil)
	if err != nil {
		log.Error("Error sending a message: ", err)
	}
}

func writeLoop(ctx context.Context, wakuNode *node.WakuNode) {
	for {
		time.Sleep(2 * time.Second)
		write(ctx, wakuNode, "Hello world!")
	}
}

func readLoop(wakuNode *node.WakuNode) {
	sub, err := wakuNode.Subscribe(&pubSubTopic)
	if err != nil {
		log.Error("Could not subscribe: ", err)
		return
	}

	for value := range sub.C {
		payload, err := node.DecodePayload(value.Message(), &node.KeyInfo{Kind: node.None})
		if err != nil {
			fmt.Println(err)
			return
		}

		log.Info("Received msg, ", string(payload.Data))
	}
}
