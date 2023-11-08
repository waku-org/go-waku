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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var log = utils.Logger().Named("rln")

// Update these values
// ============================================================================
var ethClientAddress = "wss://sepolia.infura.io/ws/v3/API_KEY_GOES_HERE"
var contractAddress = "0xF471d71E9b1455bBF4b85d475afb9BB0954A29c4"
var keystorePath = "./rlnKeystore.json"
var keystorePassword = "password"
var membershipIndex = uint(0)
var contentTopic, _ = protocol.NewContentTopic("rln", "1", "test", "proto")
var pubsubTopic = protocol.DefaultPubsubTopic{}

// ============================================================================

func main() {
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

	spamHandler := func(message *pb.WakuMessage, topic string) error {
		fmt.Println("Spam message received")
		return nil
	}

	wakuNode, err := node.New(
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
		node.WithNTP(),
		node.WithWakuRelay(),
		node.WithDynamicRLNRelay(
			keystorePath,
			keystorePassword,
			"", // Will use default tree path
			common.HexToAddress(contractAddress),
			&membershipIndex,
			spamHandler,
			ethClientAddress,
		),
	)
	if err != nil {
		log.Error("Error creating wakunode", zap.Error(err))
		return
	}

	if err := wakuNode.Start(ctx); err != nil {
		log.Error("Error starting wakunode", zap.Error(err))
		return
	}

	go writeLoop(ctx, wakuNode)
	go readLoop(ctx, wakuNode)

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

func write(ctx context.Context, wakuNode *node.WakuNode, msgContent string) {
	var version uint32 = 0

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
		Version:      proto.Uint32(version),
		ContentTopic: contentTopic.String(),
		Timestamp:    utils.GetUnixEpoch(wakuNode.Timesource()),
	}

	err = wakuNode.RLNRelay().AppendRLNProof(msg, wakuNode.Timesource().Now())
	if err != nil {
		log.Error("Error appending proof", zap.Error(err))
	}

	_, err = wakuNode.Relay().Publish(ctx, msg, relay.WithPubSubTopic(pubsubTopic.String()))
	if err != nil {
		log.Error("Error sending a message", zap.Error(err))
	}
}

func writeLoop(ctx context.Context, wakuNode *node.WakuNode) {
	for {
		time.Sleep(1 * time.Second)
		write(ctx, wakuNode, "Hello world!")
	}
}

func readLoop(ctx context.Context, wakuNode *node.WakuNode) {
	sub, err := wakuNode.Relay().Subscribe(ctx, protocol.NewContentFilter(pubsubTopic.String()))
	if err != nil {
		log.Error("Could not subscribe", zap.Error(err))
		return
	}

	for envelope := range sub[0].Ch {
		if envelope.Message().ContentTopic != contentTopic.String() {
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
