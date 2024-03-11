package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"noise/pb"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	n "github.com/waku-org/go-noise"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/noise"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var log = utils.Logger().Named("noise")

func main() {
	// Removing noisy logs
	lvl, err := logging.LevelFromString("error")
	if err != nil {
		panic(err)
	}
	logging.SetAllLoggers(lvl)

	hostAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprint("0.0.0.0:0"))

	wakuNode, err := node.New(
		node.WithHostAddress(hostAddr),
		node.WithNTP(),
		node.WithWakuRelay(),
	)
	if err != nil {
		log.Error("could not instantiate waku", zap.Error(err))
		return
	}

	if err := wakuNode.Start(context.Background()); err != nil {
		log.Error("could not start waku", zap.Error(err))
		return
	}

	discoverFleetNodes(wakuNode)

	myStaticKey, _ := n.DH25519.GenerateKeypair()
	myEphemeralKey, _ := n.DH25519.GenerateKeypair()

	relayMessenger, err := noise.NewWakuRelayMessenger(context.Background(), wakuNode.Relay(), nil, timesource.NewDefaultClock())
	if err != nil {
		log.Error("could not create relay messenger", zap.Error(err))
		return
	}

	pairingObj, err := noise.NewPairing(myStaticKey, myEphemeralKey, noise.WithDefaultResponderParameters(), relayMessenger, utils.Logger())
	if err != nil {
		log.Error("could not create pairing object", zap.Error(err))
		return
	}

	qrString, qrMessageNameTag := pairingObj.PairingInfo()
	qrURL := "messageNameTag=" + url.QueryEscape(hex.EncodeToString(qrMessageNameTag[:])) + "&qrCode=" + url.QueryEscape(qrString)

	wg := sync.WaitGroup{}

	// Execute in separate go routine
	wg.Add(1)
	go func() {
		defer wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		err := pairingObj.Execute(ctx)
		if err != nil {
			log.Error("could not perform handshake", zap.Error(err))
			return
		}
	}()

	// Confirmation is done by manually
	wg.Add(1)
	go func() {
		defer wg.Done()
		authCode := <-pairingObj.AuthCode()
		fmt.Println("=============================================")
		fmt.Println("TODO: ask to confirm pairing. Automaticaly confirming authcode", authCode)
		fmt.Println("=============================================")
		err := pairingObj.ConfirmAuthCode(true)
		if err != nil {
			log.Error("could not confirm authcode", zap.Error(err))
			return
		}
	}()

	fmt.Println("=============================================================================")
	fmt.Printf("Browse https://examples.waku.org/noise-js/?%s\n", qrURL)
	fmt.Println("=============================================================================")

	wg.Wait()

	// Securely transferring messages
	if pairingObj.HandshakeComplete() {
		go writeLoop(context.Background(), wakuNode, pairingObj)
		go readLoop(context.Background(), wakuNode, pairingObj)
	}

	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\n\n\nReceived signal, shutting down...")

	// shut the node down
	wakuNode.Stop()

}

func discoverFleetNodes(wakuNode *node.WakuNode) error {
	log.Info("Connecting to test fleet...")

	dnsDiscoveryUrl := "enrtree://AOGYWMBYOUIMOENHXCHILPKY3ZRFEULMFI4DOM442QSZ73TT2A7VI@test.waku.nodes.status.im"
	nodes, err := dnsdisc.RetrieveNodes(context.Background(), dnsDiscoveryUrl)
	if err != nil {
		return err
	}
	var peerInfo []peer.AddrInfo
	for _, n := range nodes {
		peerInfo = append(peerInfo, n.PeerInfo)
	}

	wg := sync.WaitGroup{}
	wg.Add(len(peerInfo))
	for _, p := range peerInfo {
		go func(p peer.AddrInfo) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
			defer cancel()
			err = wakuNode.DialPeerWithInfo(ctx, p)
			if err != nil {
				log.Error("could not connect", zap.String("peerID", p.ID.String()), zap.Error(err))
			} else {
				log.Info("connected", zap.String("peerID", p.ID.String()))
			}
		}(p)

	}
	wg.Wait()
	return nil
}

func writeLoop(ctx context.Context, wakuNode *node.WakuNode, pairingObj *noise.Pairing) {
	cnt := 0
	for {
		time.Sleep(4 * time.Second)

		cnt++
		chatMessage := &pb.Chat2Message{
			Timestamp: uint64(wakuNode.Timesource().Now().Unix()),
			Nick:      "go-waku",
			Payload:   []byte(fmt.Sprintf("Hello World! #%d", cnt)),
		}

		chatMessageBytes, err := proto.Marshal(chatMessage)
		if err != nil {
			log.Error("Error encoding message: ", zap.Error(err))
			continue
		}

		msg, err := pairingObj.Encrypt(chatMessageBytes)
		if err != nil {
			log.Error("error encrypting message", zap.Error(err))
			continue
		}

		msg.Timestamp = utils.GetUnixEpoch(wakuNode.Timesource())

		_, err = wakuNode.Relay().Publish(ctx, msg, relay.WithDefaultPubsubTopic())
		if err != nil {
			log.Error("Error sending a message", zap.Error(err))
		}
	}
}

func readLoop(ctx context.Context, wakuNode *node.WakuNode, pairingObj *noise.Pairing) {
	sub, err := wakuNode.Relay().Subscribe(ctx, protocol.NewContentFilter(relay.DefaultWakuTopic))
	if err != nil {
		log.Error("Could not subscribe", zap.Error(err))
		return
	}

	for value := range sub[0].Ch {
		if value.Message().ContentTopic != pairingObj.ContentTopic {
			continue
		}

		msgBytes, err := pairingObj.Decrypt(value.Message())
		if err != nil {
			log.Debug("Error decoding a message", zap.Error(err))
			continue
		}

		msg := &pb.Chat2Message{}
		if err := proto.Unmarshal(msgBytes, msg); err != nil {
			log.Error("Error decoding a message", zap.Error(err))
			continue
		}

		fmt.Println("Received msg from ", msg.Nick, " - "+string(msg.Payload))
	}
}
