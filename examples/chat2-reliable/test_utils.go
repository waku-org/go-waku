package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/node"
)

type TestNetworkController struct {
	nodes []*node.WakuNode
	chats []*Chat
	mu    sync.Mutex
	ctx   context.Context
}

func NewNetworkController(ctx context.Context, nodes []*node.WakuNode, chats []*Chat) *TestNetworkController {
	return &TestNetworkController{
		nodes: nodes,
		chats: chats,
		ctx:   ctx,
	}
}

func (nc *TestNetworkController) DisconnectNode(node *node.WakuNode) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	for _, other := range nc.nodes {
		if node != other {
			nc.disconnectPeers(node.Host(), other.Host())
		}
	}
}

func (nc *TestNetworkController) ReconnectNode(node *node.WakuNode) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	for _, other := range nc.nodes {
		if node != other && !nc.IsConnected(node, other) {
			nc.connectPeers(node.Host(), other.Host())
			fmt.Printf("Reconnected node %s to node %s\n", node.Host().ID().String(), other.Host().ID().String())
		}
	}
}

func (nc *TestNetworkController) disconnectPeers(h1, h2 host.Host) {
	h1.Network().ClosePeer(h2.ID())
	h2.Network().ClosePeer(h1.ID())
}

func (nc *TestNetworkController) connectPeers(h1, h2 host.Host) {
	_, err := h1.Network().DialPeer(nc.ctx, h2.ID())
	if err != nil {
		fmt.Printf("Error connecting peers: %v\n", err)
	}
}

func (nc *TestNetworkController) IsConnected(n1, n2 *node.WakuNode) bool {
	peerID, err := peer.Decode(n2.ID())
	if err != nil {
		fmt.Printf("Error decoding peer ID: %v\n", err)
		return false
	}
	return n1.Host().Network().Connectedness(peerID) == network.Connected
}
