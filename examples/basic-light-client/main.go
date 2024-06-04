package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	cli "github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/payload"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var log = utils.Logger().Named("basic-light-client")

var ClusterID = altsrc.NewUintFlag(&cli.UintFlag{
	Name:        "cluster-id",
	Value:       1,
	Usage:       "Cluster id that the client is interested in connecting to.",
	Destination: &clusterID,
})

var Shard = altsrc.NewUintFlag(&cli.UintFlag{
	Name:        "shard",
	Value:       0,
	Usage:       "shard that the node is interested in publishing/receiving from.",
	Destination: &shard,
})

var StaticNode = altsrc.NewStringFlag(&cli.StringFlag{
	Name:        "maddr",
	Usage:       "multiaddress of static node to connect to.",
	Destination: &multiaddress,
	Required:    true,
})

var clusterID, shard uint
var pubsubTopicStr string
var multiaddress string

func main() {

	cliFlags := []cli.Flag{
		ClusterID,
		Shard,
		StaticNode,
	}

	app := &cli.App{
		Name:  "basic-light-client-example",
		Flags: cliFlags,
		Action: func(c *cli.Context) error {
			err := Execute()
			if err != nil {
				utils.Logger().Error("failure while executing wakunode", zap.Error(err))
				switch e := err.(type) {
				case cli.ExitCoder:
					return e
				case error:
					return cli.Exit(err.Error(), 1)
				}
			}
			return nil
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}

}

func Execute() error {

	var cTopic, err = protocol.NewContentTopic("basic-light-client", "1", "test", "proto")
	if err != nil {
		return errors.New("invalid contentTopic")
	}
	contentTopic := cTopic.String()
	hostAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	key, err := randomHex(32)
	if err != nil {
		log.Error("Could not generate random key", zap.Error(err))
		return err
	}
	prvKey, err := crypto.HexToECDSA(key)
	if err != nil {
		log.Error("Could not convert hex into ecdsa key", zap.Error(err))
		return err
	}

	ctx := context.Background()
	lightNode, err := node.New(
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
		node.WithWakuFilterLightNode(),
		node.WithClusterID(uint16(clusterID)),
		//node.WithLogLevel(zapcore.DebugLevel),
	)
	if err != nil {
		log.Error("Error creating wakunode", zap.Error(err))
		return err
	}

	err = lightNode.Start(ctx)
	if err != nil {
		log.Error("Error starting wakunode", zap.Error(err))
		return err
	}

	//Populate pubsubTopic if shard is specified. Otherwise it is derived via autosharing algorithm
	if shard != 0 {
		pubsubTopic := protocol.NewStaticShardingPubsubTopic(uint16(clusterID), uint16(shard))
		pubsubTopicStr = pubsubTopic.String()
	}

	maddr, err := multiaddr.NewMultiaddr(multiaddress)
	if err != nil {
		log.Info("Error decoding multiaddr ", zap.Error(err))
	}
	peerID, err := lightNode.AddPeer(maddr, wps.Static,
		[]string{pubsubTopicStr}, filter.FilterSubscribeID_v20beta1, lightpush.LightPushID_v20beta1)
	if err != nil {
		log.Info("Error adding filter peer on light node ", zap.Error(err))
	}

	useFilterAndLightPush(lightNode, contentTopic, pubsubTopicStr, peerID)

	// shut the node down
	lightNode.Stop()
	return nil
}

func useFilterAndLightPush(lightNode *node.WakuNode, contentTopic string, pubsubTopic string, filterNode peer.ID) {

	// Send FilterRequest from light node to full node
	cf := protocol.ContentFilter{
		PubsubTopic:   pubsubTopic,
		ContentTopics: protocol.NewContentTopicSet(contentTopic),
	}
	time.Sleep(2 * time.Second)
	log.Info("Subscribing to peer ", zap.String("peerId", filterNode.String()))
	theFilter, err := lightNode.FilterLightnode().Subscribe(context.Background(), cf, filter.WithPeer(filterNode))
	if err != nil {
		panic(err)
	}

	go func() {
		for env := range theFilter[0].C { //Safely picking first subscriptions since only 1 contentTopic is subscribed
			log.Info("Light node received msg ", zap.String("message", string(env.Message().Payload)))
		}
		log.Info("Message channel closed!")
	}()

	time.Sleep(2 * time.Second)

	msg := &pb.WakuMessage{ContentTopic: contentTopic, Version: proto.Uint32(1), Timestamp: proto.Int64(time.Now().Unix()), Payload: []byte("Hello World!")}
	hash, err := lightNode.Lightpush().Publish(context.Background(), msg, lightpush.WithPeer(filterNode), lightpush.WithPubSubTopic(pubsubTopic))
	if err != nil {
		panic(err)
	}
	log.Info("published msg via lightpush with hash:", zap.Stringer("hash", hash))

	log.Info("Done sending msgs.......")

	log.Info("Press Ctrl+C to exit safely")

	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\n\n\nReceived signal, shutting down...")

	log.Info("UnSubscribing to peer ", zap.String("filterNode", filterNode.String()))

	result, err := lightNode.FilterLightnode().Unsubscribe(context.Background(), cf, filter.WithPeer(filterNode))
	if err != nil {
		log.Error("failed to unsubscribe due to ", zap.Error(err))
		panic(err)
	}
	for _, err := range result.Errors() {
		if err.Err != nil {
			log.Error("failed to unsubscribe due to res err", zap.Error(err.Err))
			panic(err)
		}
	}

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
		ContentTopic: contentTopic,
		Timestamp:    utils.GetUnixEpoch(wakuNode.Timesource()),
	}

	_, err = wakuNode.Relay().Publish(ctx, msg, relay.WithPubSubTopic(pubsubTopicStr))
	if err != nil {
		log.Error("Error sending a message", zap.Error(err))
	}
	log.Info("Published msg,", zap.String("data", string(msg.Payload)))
}

func writeLoop(ctx context.Context, wakuNode *node.WakuNode, contentTopic string) {
	for {
		time.Sleep(2 * time.Second)
		write(ctx, wakuNode, contentTopic, "Hello world!")
	}
}

func readLoop(ctx context.Context, wakuNode *node.WakuNode, contentTopic string) {
	sub, err := wakuNode.Relay().Subscribe(ctx, protocol.NewContentFilter(pubsubTopicStr, contentTopic))
	if err != nil {
		log.Error("Could not subscribe", zap.Error(err))
		return
	}

	for envelope := range sub[0].Ch {
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
