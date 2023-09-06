package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	bootNode, err := bootnode(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}

	time.Sleep(3 * time.Second)

	unreachableNode, err := unreachableNode(ctx, bootNode.ENR())
	if err != nil {
		fmt.Println(err)
		return
	}
	defer unreachableNode.Stop()

	err = unreachableNode.DialPeer(ctx, "/dns4/node-01.do-ams3.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAm6HZZr7aToTvEBPpiys4UxajCTU97zj5v7RNR2gbniy1D")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Wait 15 seconds because there is an initial delay in libp2p for circuit relay ...
	fmt.Println("==============================================================================")
	fmt.Println("Waiting some seconds...")
	time.Sleep(15 * time.Second)
	fmt.Println("Waited enough :D")
	fmt.Println("==============================================================================")

	px, err := peerExchangeClient(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = px.DialPeerWithMultiAddress(ctx, bootNode.ListenAddresses()[0])
	if err != nil {
		fmt.Println("ERROR DIALING BOOTNODE:", err)
		return
	}

	go func() {
		t := time.NewTicker(3 * time.Second)
		for range t.C {
			err := px.PeerExchange().Request(ctx, 3)
			if err != nil {
				fmt.Println("==============================================================================")
				fmt.Println("COULD NOT RETRIEVE PEERS")
				fmt.Println("==============================================================================")
			} else {
				peers := px.Host().Peerstore().(peerstore.WakuPeerstore).PeersByOrigin(peerstore.PeerExchange)
				fmt.Println("==============================================================================")
				fmt.Println("Peers obtained via peer exchange: ")
				for _, p := range peers {
					hostInfo, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", p.Pretty()))
					if err != nil {
						continue
					}

					for _, a := range px.Host().Peerstore().Addrs(p) {
						addr := a.Encapsulate(hostInfo)
						fmt.Println(addr)
					}
				}
				fmt.Println("==============================================================================")

			}
		}
	}()

	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\n\n\nReceived signal, shutting down...")
}

func bootnode(ctx context.Context) (*node.WakuNode, error) {
	hostAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:45111")

	prvKey, err := crypto.HexToECDSA("1122334455667788990011223344556677889900112233445566778899001122")
	if err != nil {
		return nil, err
	}

	wakuNode, err := node.New(
		node.WithDiscoveryV5(64111, nil, true),
		node.WithPeerExchange(),
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
	)

	if err != nil {
		return nil, err
	}

	if err := wakuNode.Start(ctx); err != nil {
		return nil, err
	}

	utils.Logger().Warn("bootnode listen addresses", zap.Any("addresses", wakuNode.ListenAddresses()))
	utils.Logger().Warn("bootnode enr", zap.Stringer("enr", wakuNode.ENR()))

	if err = wakuNode.DiscV5().Start(ctx); err != nil {
		return nil, err
	}

	return wakuNode, nil
}

func unreachableNode(ctx context.Context, bootnode *enode.Node) (*node.WakuNode, error) {
	prvKey, err := crypto.HexToECDSA("1122334455667788990011223344556677889900112233445566778899003344")
	if err != nil {
		return nil, err
	}

	hostAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:45222")

	wakuNode, err := node.New(
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
		node.WithDiscoveryV5(64222, []*enode.Node{bootnode}, true),
		node.WithLibP2POptions(append(node.DefaultLibP2POptions, libp2p.EnableRelay(), libp2p.ForceReachabilityPrivate())...),
		node.WithWakuRelay(),
		node.WithSecureWebsockets("0.0.0.0", 6443, "./certs/cert.pem", "./certs/key.pem"),
	)

	if err != nil {
		return nil, err
	}

	if err := wakuNode.Start(ctx); err != nil {
		return nil, err
	}

	utils.Logger().Warn("unreachable node listen addresses", zap.Any("addresses", wakuNode.ListenAddresses()))
	utils.Logger().Warn("unreachable node enr", zap.Stringer("enr", wakuNode.ENR()))

	if err = wakuNode.DiscV5().Start(ctx); err != nil {
		return nil, err
	}

	sub, err := wakuNode.Relay().Subscribe(ctx)
	if err != nil {
		return nil, err
	}

	contentTopic, err := protocol.NewContentTopic("basic2", 1, "test", "proto")
	if err != nil {
		return nil, err
	}

	fmt.Println(contentTopic.String())

	go func() {
		for m := range sub.Ch {
			if m.Message().ContentTopic == contentTopic.String() {
				fmt.Println("MESSAGE RECEIVED: ", string(m.Message().Payload))
			}
		}
	}()

	return wakuNode, nil
}

func peerExchangeClient(ctx context.Context) (*node.WakuNode, error) {
	wakuNode, err := node.New(
		node.WithPeerExchange(),
	)

	if err != nil {
		return nil, err
	}

	if err := wakuNode.Start(ctx); err != nil {
		return nil, err
	}

	return wakuNode, nil
}
