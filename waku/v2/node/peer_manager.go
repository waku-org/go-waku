package node

import (
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peerstore"
)

type Connectedness string

const (
	// NotConnected: default state for a new peer. No connection and no further information on connectedness.
	NotConnected Connectedness = "NotConnected"
	// CannotConnect: attempted to connect to peer, but failed.
	CannotConnect Connectedness = "CannotConnect"
	// CanConnect: was recently connected to peer and disconnected gracefully.
	CanConnect Connectedness = "CanConnect"
	// Connected: actively connected to peer.
	Connected Connectedness = "Connected"
)

/*
type
  ConnectionBook* = object of PeerBook[Connectedness]
*/

type WakuPeerStore struct {
	connectionBook peerstore.Peerstore
}

type PeerManager struct {
	sw        host.Host
	peerStore *WakuPeerStore
}

func NewWakuPeerStore() *WakuPeerStore {
	p := new(WakuPeerStore)
	return p
}

func NewPeerManager(sw host.Host) *PeerManager {
	peerStore := NewWakuPeerStore()
	p := new(PeerManager)
	p.sw = sw
	p.peerStore = peerStore
	return p
}
