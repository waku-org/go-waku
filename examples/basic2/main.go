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
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

var log = utils.Logger().Named("basic2")

func main() {
	var cTopic, err = protocol.NewContentTopic("basic2", 1, "test", "proto")
	if err != nil {
		fmt.Println("Invalid contentTopic")
		return
	}
	contentTopic := cTopic.String()

	hostAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	key, err := randomHex(32)
	if err != nil {
		log.Error("Could not generate random key", zap.Error(err))
		return
	}
	prvKey, err := crypto.HexToECDSA(key)
	if err != nil {
		log.Error("Could not convert hex into ecdsa key", zap.Error(err))
		return
	}

	ctx := context.Background()

	wakuNode, err := node.New(
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
		node.WithNTP(),
		node.WithWakuRelay(),
	)
	if err != nil {
		log.Error("Error creating wakunode", zap.Error(err))
		return
	}

	if err := wakuNode.Start(ctx); err != nil {
		log.Error("Error starting wakunode", zap.Error(err))
		return
	}

	go writeLoop(ctx, wakuNode, contentTopic)
	go readLoop(ctx, wakuNode, contentTopic)

	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\n\n\nReceived signal, shutting down...")

	// shut the node down
	wakuNode.Stop()

}

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func write(ctx context.Context, wakuNode *node.WakuNode, contentTopic string, msgContent string) {
	var version uint32 = 0
	var timestamp int64 = utils.GetUnixEpoch(wakuNode.Timesource())

	p := new(payload.Payload)
	p.Data = []byte(wakuNode.ID() + ": " + msgContent)
	p.Key = &payload.KeyInfo{Kind: payload.None}

	payload, err := p.Encode(version)
	if err != nil {
		log.Error("Error encoding the payload", zap.Error(err))
		return
	}

	msg := &pb.WakuMessage{
		Payload:      payload,
		Version:      version,
		ContentTopic: contentTopic,
		Timestamp:    timestamp,
	}

	_, err = wakuNode.Relay().Publish(ctx, msg)
	if err != nil {
		log.Error("Error sending a message", zap.Error(err))
	}
}

func writeLoop(ctx context.Context, wakuNode *node.WakuNode, contentTopic string) {
	for {
		time.Sleep(2 * time.Second)
		write(ctx, wakuNode, contentTopic, "Hello world!")
	}
}

func readLoop(ctx context.Context, wakuNode *node.WakuNode, contentTopic string) {
	sub, err := wakuNode.Relay().Subscribe(ctx)
	if err != nil {
		log.Error("Could not subscribe", zap.Error(err))
		return
	}

	for envelope := range sub.Ch {
		if envelope.Message().ContentTopic != contentTopic {
			continue
		}

		payload, err := payload.DecodePayload(envelope.Message(), &payload.KeyInfo{Kind: payload.None})
		if err != nil {
			log.Error("Error decoding payload", zap.Error(err))
			continue
		}

		log.Info("Received msg, ", zap.String("data", string(payload.Data)))
	}
}
