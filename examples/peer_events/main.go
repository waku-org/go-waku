package main

import (
	"context"
	"crypto/ecdsa"
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
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"

	//"github.com/status-im/go-waku/waku/v2/protocol/store"
	"github.com/status-im/go-waku/waku/v2/utils"
)

var log = logging.Logger("peer_events")

var pubSubTopic = relay.DefaultWakuTopic

const contentTopic = "test"

func main() {
	lvl, err := logging.LevelFromString("info")
	if err != nil {
		panic(err)
	}
	logging.SetAllLoggers(lvl)

	type AddrAndKey struct {
		addr *net.TCPAddr
		key  *ecdsa.PrivateKey
	}
	nodeCount := 4
	addrsAndKeys := make([]*AddrAndKey, nodeCount)

	for i := 0; i < nodeCount; i++ {
		addr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("0.0.0.0:6000%d", i))
		key, err := randomHex(32)
		if err != nil {
			log.Error("Could not generate random key")
			return
		}
		prvKey, err := crypto.HexToECDSA(key)
		addrAndKey := &AddrAndKey{
			addr: addr,
			key:  prvKey,
		}
		addrsAndKeys[i] = addrAndKey
	}

	ctx := context.Background()

	//connStatusChan := make(chan node.ConnStatus)
	log.Info("### create relayNode1")
	relayNode1, err := node.New(ctx,
		node.WithPrivateKey(addrsAndKeys[0].key),
		node.WithHostAddress([]net.Addr{addrsAndKeys[0].addr}),
		node.WithWakuRelay(),
		//node.WithConnStatusChan(connStatusChan),
		node.WithWakuStore(true),
		node.WithKeepAlive(time.Duration(2)*time.Second),
	)

	// relayNode2, err := node.New(ctx,
	// 	node.WithPrivateKey(addrsAndKeys[1].key),
	// 	node.WithHostAddress([]net.Addr{addrsAndKeys[1].addr}),
	// 	node.WithWakuRelay(),
	// )

	log.Info("### before DialPeer")
	//staticNode := "/ip4/8.210.222.231/tcp/30303/p2p/16Uiu2HAm4v86W3bmT1BiH6oSPzcsSr24iDQpSN5Qa992BCjjwgrD"
	staticNode := "/ip4/188.166.135.145/tcp/30303/p2p/16Uiu2HAmL5okWopX7NqZWBUKVqW8iUxCEmd5GMHLVPwCgzYzQv3e"
	relayNode1.DialPeer(staticNode)
	//relayNode2.DialPeer(relayNode1.ListenAddresses()[0])

	//go writeLoop(ctx, relayNode1)
	//go readLoop(relayNode1)

	//go readLoop(relayNode2)

	log.Info("### Peer dialled")
	// printNodeConns := func(node *node.WakuNode) {
	// 	log.Info(node.Host().ID(), ": ", "peerCount: ", len(node.Host().Peerstore().Peers()))
	// 	log.Info("node peers: ")
	// 	for k, v := range node.GetPeerStats() {
	// 		log.Info(k, " ", v)
	// 	}
	// 	log.Info(node.Host().ID(), ": ", "isOnline/hasHistory ", node.IsOnline(), " ", node.HasHistory())
	// 	log.Info("end")

	// }

	// go func() {
	// 	for connStatus := range connStatusChan {
	// 		log.Info("Conn status update: ", connStatus)
	// 	}
	// }()
	// go func() {
	// 	ticker := time.NewTicker(time.Millisecond * 1000)
	// 	defer ticker.Stop()
	// 	for {
	// 		select {
	// 		case <-ticker.C:
	// 			printNodeConns(relayNode1)
	// 		}
	// 	}
	// }()

	// time.Sleep(3 * time.Second)
	// log.Info("stop relayNode2")
	// relayNode2.Host().Close()

	// time.Sleep(3 * time.Second)

	//	log.Info("start relayNode3")

	//	relayNode3, err := node.New(ctx,
	//		node.WithPrivateKey(addrsAndKeys[2].key),
	//		node.WithHostAddress([]net.Addr{addrsAndKeys[2].addr}),
	//		node.WithWakuRelay(),
	//	)

	//	relayNode3.DialPeer(relayNode1.ListenAddresses()[0])

	//	time.Sleep(3 * time.Second)
	//	log.Info("stop relayNode3")
	//	//relayNode3.Stop()

	//	log.Info("start storeNode")
	//	// Start a store node
	//	storeNode, _ := node.New(ctx,
	//		node.WithPrivateKey(addrsAndKeys[3].key),
	//		node.WithHostAddress([]net.Addr{addrsAndKeys[3].addr}),
	//		node.WithWakuRelay(),
	//		node.WithWakuStore(true),
	//	)
	//	tCtx, _ := context.WithTimeout(ctx, 5*time.Second)
	//	log.Info("#before AddStorePeer")
	//	storeNodeId, err := relayNode1.AddStorePeer(storeNode.ListenAddresses()[0])
	//	time.Sleep(3 * time.Second)
	//	log.Info("#before Query")
	//	_, err = relayNode1.Query(tCtx, []string{contentTopic}, 0, 0, store.WithPeer(*storeNodeId))
	//	log.Info("storeNode.ListenAddresses(): ", storeNode.ListenAddresses(), storeNodeId)
	//	if err != nil {
	//		log.Info("### error adding store peer: ", err)
	//	}

	//	time.Sleep(3 * time.Second)
	//	log.Info("stop storeNode")
	//	storeNode.Stop()

	//	time.Sleep(3 * time.Second)
	//	// // Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\n\n\nReceived signal, shutting down...")

	//	// shut the nodes down
	//	relayNode1.Stop()

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
	var timestamp float64 = utils.GetUnixEpoch()

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
		//log.Info("peerCount: ", len(wakuNode.Host().Peerstore().Peers()))
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

		log.Info("Received msg, ", wakuNode.ID(), ", payload: ", string(payload.Data))
	}
}
