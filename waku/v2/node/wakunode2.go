package node

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"

	"github.com/status-im/go-waku/waku/v2/protocol"
)

// Default clientId
const clientId string = "Go Waku v2 node"

// XXX: Weird type, should probably be using pubsub Topic object name?
type Topic string

type Message []byte

type WakuInfo struct {
	// NOTE One for simplicity, can extend later as needed
	listenStr        string
	multiaddrStrings []byte
}

type MessagePair struct {
	a *Topic
	b *protocol.WakuMessage
}

// NOTE based on Eth2Node in NBC eth2_network.nim
type WakuNode struct {
	peerManager *PeerManager
	Host        host.Host
	// wakuRelay *WakuRelay
	// wakuStore *WakuStore
	// wakuFilter *WakuFilter
	//wakuSwap *WakuSwap
	//wakuRlnRelay *WakuRLNRelay
	peerInfo peer.AddrInfo
	// libp2pTransportLoops []Future[void] ??
	// TODO Revisit messages field indexing as well as if this should be Message or WakuMessage
	messages []MessagePair
	//filters *Filters
	subscriptions protocol.MessageNotificationSubscriptions
	// rng *BrHmacDrbgContext // ???
}

// Public API
//

func New(ctx context.Context, nodeKey crypto.PrivKey, hostAddr net.Addr, extAddr net.Addr) (*WakuNode, error) {
	// Creates a Waku Node.
	if hostAddr == nil {
		return nil, errors.New("Host address cannot be null")
	}

	var multiAddresses []ma.Multiaddr
	hostAddrMA, err := manet.FromNetAddr(hostAddr)
	if err != nil {
		return nil, err
	}
	multiAddresses = append(multiAddresses, hostAddrMA)

	if extAddr != nil {
		extAddrMA, err := manet.FromNetAddr(extAddr)
		if err != nil {
			return nil, err
		}
		multiAddresses = append(multiAddresses, extAddrMA)
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrs(multiAddresses...),
		libp2p.Identity(nodeKey),
		libp2p.DefaultTransports,  //
		libp2p.NATPortMap(),       // Attempt to open ports using uPNP for NATed hosts.
		libp2p.DisableRelay(),     // TODO: what is this?
		libp2p.EnableNATService(), // TODO: what is this?
	}

	host, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	w := new(WakuNode)
	w.peerManager = NewPeerManager(host)
	w.Host = host
	// w.filters = new(Filters)

	hostInfo, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", host.ID().Pretty()))
	for _, addr := range host.Addrs() {
		fullAddr := addr.Encapsulate(hostInfo)
		log.Printf("Listening on %s\n", fullAddr)
	}

	return w, nil
}

func (node *WakuNode) Stop() error {
	// TODO:
	//if not node.wakuRelay.isNil:
	//  await node.wakuRelay.stop()

	return node.Host.Close()
}
