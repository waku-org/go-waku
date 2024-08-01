package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/node"
)

type NetworkController struct {
	nodes     []*node.WakuNode
	chats     []*Chat
	connected map[peer.ID]map[peer.ID]bool
	mu        sync.Mutex
}

func NewNetworkController(nodes []*node.WakuNode, chats []*Chat) *NetworkController {
	nc := &NetworkController{
		nodes:     nodes,
		chats:     chats,
		connected: make(map[peer.ID]map[peer.ID]bool),
	}

	for _, n := range nodes {
		nc.connected[n.Host().ID()] = make(map[peer.ID]bool)
		for _, other := range nodes {
			if n != other {
				nc.connected[n.Host().ID()][other.Host().ID()] = true
			}
		}
	}

	return nc
}

func (nc *NetworkController) DisconnectNode(node *node.WakuNode) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	nodeID := node.Host().ID()
	for _, other := range nc.nodes {
		otherID := other.Host().ID()
		if nodeID != otherID {
			nc.disconnectPeers(node.Host(), other.Host())
			nc.connected[nodeID][otherID] = false
			nc.connected[otherID][nodeID] = false
		}
	}

	// Set the node as disconnected in the Chat
	for _, chat := range nc.chats {
		if chat.node == node {
			chat.SetDisconnected(true)
			break
		}
	}
}

func (nc *NetworkController) ReconnectNode(node *node.WakuNode) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	nodeID := node.Host().ID()
	for _, other := range nc.nodes {
		otherID := other.Host().ID()
		if nodeID != otherID && !nc.connected[nodeID][otherID] {
			nc.connectPeers(node.Host(), other.Host())
			nc.connected[nodeID][otherID] = true
			nc.connected[otherID][nodeID] = true
			fmt.Printf("Reconnected node %s to node %s\n", nodeID.String(), otherID.String())
		}
	}

	// Set the node as connected in the Chat
	for _, chat := range nc.chats {
		if chat.node == node {
			chat.SetDisconnected(false)
			fmt.Printf("Set node %s as connected in Chat\n", nodeID.String())
			break
		}
	}
}

func (nc *NetworkController) disconnectPeers(h1, h2 host.Host) {
	h1.Network().ClosePeer(h2.ID())
	h2.Network().ClosePeer(h1.ID())
}

func (nc *NetworkController) connectPeers(h1, h2 host.Host) {
	ctx := context.Background()
	h1.Connect(ctx, peer.AddrInfo{ID: h2.ID(), Addrs: h2.Addrs()})
}

func (nc *NetworkController) IsConnected(n1, n2 *node.WakuNode) bool {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	return nc.connected[n1.Host().ID()][n2.Host().ID()]
}

// func (c *Chat) checkPeerConnections() {
// 	peers := c.node.Host().Network().Peers()
// 	fmt.Printf("Node %s: Connected to %d peers\n", c.node.Host().ID().String(), len(peers))
// 	for _, peer := range peers {
// 		fmt.Printf("  - %s\n", peer.String())
// 	}
// }
