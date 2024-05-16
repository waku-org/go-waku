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
	logging "github.com/ipfs/go-log/v2"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/payload"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"google.golang.org/protobuf/proto"
)

var log = logging.Logger("filter2")

var pubSubTopic = protocol.DefaultPubsubTopic{}

const contentTopic = "/filter2test/1/testTopic/proto"

func main() {
	hostAddr1, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:60000")
	hostAddr2, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:60001")

	key1, err := randomHex(32)
	if err != nil {
		log.Error("Could not generate random key")
		return
	}
	prvKey1, err := crypto.HexToECDSA(key1)
	if err != nil {
		log.Error("Invalid key")
		return
	}

	key2, err := randomHex(32)
	if err != nil {
		log.Error("Could not generate random key")
		return
	}

	prvKey2, err := crypto.HexToECDSA(key2)
	if err != nil {
		log.Error("Invalid key")
		return
	}

	ctx := context.Background()

	fullNode, err := node.New(
		node.WithPrivateKey(prvKey1),
		node.WithHostAddress(hostAddr1),
		node.WithWakuRelay(),
		node.WithWakuFilterFullNode(),
		node.WithLightPush(lightpush.WithRateLimiter(1, 1)),
	)
	if err != nil {
		panic(err)
	}

	err = fullNode.Start(ctx)
	if err != nil {
		panic(err)
	}

	lightNode, err := node.New(
		node.WithPrivateKey(prvKey2),
		node.WithHostAddress(hostAddr2),
		node.WithWakuFilterLightNode(),
	)
	if err != nil {
		panic(err)
	}

	err = lightNode.Start(ctx)
	if err != nil {
		panic(err)
	}

	//
	// Setup filter
	//

	_, err = lightNode.AddPeer(fullNode.ListenAddresses()[0], wps.Static,
		[]string{pubSubTopic.String()}, filter.FilterSubscribeID_v20beta1)
	if err != nil {
		log.Info("Error adding filter peer on light node ", err)
	}

	// Send FilterRequest from light node to full node
	cf := protocol.ContentFilter{
		PubsubTopic:   relay.DefaultWakuTopic,
		ContentTopics: protocol.NewContentTopicSet(contentTopic),
	}

	theFilter, err := lightNode.FilterLightnode().Subscribe(ctx, cf)
	if err != nil {
		panic(err)
	}

	go func() {
		for env := range theFilter[0].C { //Safely picking first subscriptions since only 1 contentTopic is subscribed
			log.Info("Light node received msg, ", string(env.Message().Payload))
		}
		log.Info("Message channel closed!")
	}()

	go writeLoop(ctx, fullNode)
	go readLoop(ctx, fullNode)
	go writeLightpushLoop(ctx, lightNode)

	go func() {
		// Unsubscribe filter after 5 seconds
		time.Sleep(5 * time.Second)
		lightNode.FilterLightnode().Unsubscribe(ctx, cf)
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
	msg := createMessage(wakuNode, msgContent)

	_, err := wakuNode.Relay().Publish(ctx, msg, relay.WithPubSubTopic(pubSubTopic.String()))
	if err != nil {
		log.Error("Error sending a message: ", err)
	}
}

func writeLightpush(ctx context.Context, wakuNode *node.WakuNode, msgContent string) {
	msg := createMessage(wakuNode, msgContent)

	_, err := wakuNode.Lightpush().Publish(ctx, msg, lightpush.WithPubSubTopic(pubSubTopic.String()))
	if err != nil {
		log.Error("Error sending a LP message: ", err)
	}
}

func createMessage(wakuNode *node.WakuNode, msgContent string) *pb.WakuMessage {
	var version uint32 = 0

	p := new(payload.Payload)
	p.Data = []byte(wakuNode.ID() + ": " + msgContent)
	p.Key = &payload.KeyInfo{Kind: payload.None}

	payload, _ := p.Encode(version)

	msg := &pb.WakuMessage{
		Payload:      payload,
		Version:      proto.Uint32(version),
		ContentTopic: contentTopic,
		Timestamp:    utils.GetUnixEpoch(wakuNode.Timesource()),
	}

	return msg
}

func writeLoop(ctx context.Context, wakuNode *node.WakuNode) {
	for {
		time.Sleep(2 * time.Second)
		write(ctx, wakuNode, "Hello world!")
	}
}

func writeLightpushLoop(ctx context.Context, wakuNode *node.WakuNode) {
	for {
		//Change this sleep value to lower than 1 second to get Lightpush rate limiting kick in
		time.Sleep(1000 * time.Millisecond)
		writeLightpush(ctx, wakuNode, "Hello World via Lightpush!")
	}
}

func readLoop(ctx context.Context, wakuNode *node.WakuNode) {
	pubsubTopic := pubSubTopic.String()
	sub, err := wakuNode.Relay().Subscribe(ctx, protocol.NewContentFilter(pubsubTopic))
	if err != nil {
		log.Error("Could not subscribe: ", err)
		return
	}

	for value := range sub[0].Ch {
		payload, err := payload.DecodePayload(value.Message(), &payload.KeyInfo{Kind: payload.None})
		if err != nil {
			fmt.Println(err)
			return
		}

		log.Info("Received msg, ", string(payload.Data))
	}
}
