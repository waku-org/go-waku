package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	n "github.com/waku-org/go-noise"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/noise"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var log = logging.Logger("noise")

func main() {
	// Removing noisy logs
	lvl, err := logging.LevelFromString("error")
	if err != nil {
		panic(err)
	}
	logging.SetAllLoggers(lvl)

	hostAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprint("0.0.0.0:0"))

	wakuNode, err := node.New(context.Background(),
		node.WithHostAddress(hostAddr),
		node.WithNTP(),
		node.WithWakuRelay(),
	)
	if err != nil {
		log.Error(err)
		return
	}

	if err := wakuNode.Start(); err != nil {
		log.Error(err)
		return
	}

	discoverFleetNodes(wakuNode)

	myStaticKey, _ := n.DH25519.GenerateKeypair()
	myEphemeralKey, _ := n.DH25519.GenerateKeypair()

	relayMessenger, err := noise.NewWakuRelayMessenger(context.Background(), wakuNode.Relay(), nil, timesource.NewDefaultClock())
	if err != nil {
		log.Error(err)
		return
	}

	pairingObj, err := noise.NewPairing(myStaticKey, myEphemeralKey, noise.WithDefaultResponderParameters(), relayMessenger, utils.Logger())
	if err != nil {
		log.Error(err)
		return
	}

	qrString, qrMessageNameTag := pairingObj.PairingInfo()
	qrURL := url.QueryEscape(hex.EncodeToString(qrMessageNameTag[:]) + ":" + qrString)

	wg := sync.WaitGroup{}

	// Execute in separate go routine
	wg.Add(1)
	go func() {
		defer wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		err := pairingObj.Execute(ctx)
		if err != nil {
			log.Error(err)
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
			log.Error(err)
			return
		}
	}()

	fmt.Println("=============================================================================")
	fmt.Printf("Browse http://localhost:8080/?%s\n", qrURL)
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

	dnsDiscoveryUrl := "enrtree://AOGECG2SPND25EEFMAJ5WF3KSGJNSGV356DSTL2YVLLZWIV6SAYBM@test.waku.nodes.status.im"
	nodes, err := dnsdisc.RetrieveNodes(context.Background(), dnsDiscoveryUrl)
	if err != nil {
		return err
	}
	var nodeList []multiaddr.Multiaddr
	for _, n := range nodes {
		nodeList = append(nodeList, n.Addresses...)
	}

	log.Info(fmt.Sprintf("Discovered and connecting to %v ", nodeList))

	wg := sync.WaitGroup{}
	wg.Add(len(nodeList))
	for _, n := range nodeList {
		go func(addr multiaddr.Multiaddr) {
			defer wg.Done()

			peerID, err := addr.ValueForProtocol(multiaddr.P_P2P)
			if err != nil {
				log.Error("error obtaining peerID", zap.Error(err))
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
			defer cancel()
			err = wakuNode.DialPeerWithMultiAddress(ctx, addr)
			if err != nil {
				log.Error("could not connect", zap.String("peerID", peerID), zap.Error(err))
			} else {
				log.Info("Connected", zap.String("peerID", peerID))
			}
		}(n)

	}
	wg.Wait()
	return nil
}

func writeLoop(ctx context.Context, wakuNode *node.WakuNode, pairingObj *noise.Pairing) {
	for {
		time.Sleep(4 * time.Second)

		chatMessage := &Chat2Message{
			Timestamp: uint64(wakuNode.Timesource().Now().Unix()),
			Nick:      "go-waku",
			Payload:   []byte("Hello World!"),
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

		msg.Timestamp = wakuNode.Timesource().Now().UnixNano()

		_, err = wakuNode.Relay().Publish(ctx, msg)
		if err != nil {
			log.Error("Error sending a message", zap.Error(err))
		}
	}
}

func readLoop(ctx context.Context, wakuNode *node.WakuNode, pairingObj *noise.Pairing) {
	sub, err := wakuNode.Relay().Subscribe(ctx)
	if err != nil {
		log.Error("Could not subscribe: ", err)
		return
	}

	for value := range sub.C {
		if value.Message().ContentTopic != pairingObj.ContentTopic {
			continue
		}

		msgBytes, err := pairingObj.Decrypt(value.Message())
		if err != nil {
			log.Debug("Error decoding a message", zap.Error(err))
			continue
		}

		msg := &Chat2Message{}
		if err := proto.Unmarshal(msgBytes, msg); err != nil {
			log.Error("Error decoding a message", zap.Error(err))
			continue
		}

		fmt.Println("Received msg from ", msg.Nick, " - "+string(msg.Payload))
	}
}
