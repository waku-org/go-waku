package peermanager

import (
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

type peerMap struct {
	mu sync.RWMutex
	m  map[peer.ID]struct{}
}

func newPeerMap() *peerMap {
	return &peerMap{
		m: map[peer.ID]struct{}{},
	}
}

func (pm *peerMap) getRandom() (peer.ID, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for pID := range pm.m {
		return pID, nil
	}
	return "", utils.ErrNoPeersAvailable

}

func (pm *peerMap) remove(pID peer.ID) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.m, pID)
}
func (pm *peerMap) add(pID peer.ID) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.m[pID] = struct{}{}
}

// ServiceSlot is for storing service slots for a given protocol topic
type ServiceSlot struct {
	mu sync.Mutex
	m  map[protocol.ID]*peerMap
}

// NewServiceSlot is a constructor for ServiceSlot
func NewServiceSlot() *ServiceSlot {
	return &ServiceSlot{
		m: map[protocol.ID]*peerMap{},
	}
}

// getPeers for getting all the peers for a given protocol
// since peerMap is only used in peerManager that's why it is unexported
func (slots *ServiceSlot) getPeers(proto protocol.ID) *peerMap {
	slots.mu.Lock()
	defer slots.mu.Unlock()
	if slots.m[proto] == nil {
		slots.m[proto] = newPeerMap()
	}
	return slots.m[proto]
}

// RemovePeer for removing peer ID for a given protocol
func (slots *ServiceSlot) RemovePeer(peerID peer.ID) {
	slots.mu.Lock()
	defer slots.mu.Unlock()
	for _, m := range slots.m {
		m.remove(peerID)
	}
}
