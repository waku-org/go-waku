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

func (pm *peerMap) GetRandom() (peer.ID, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for pId := range pm.m {
		return pId, nil
	}
	return "", utils.ErrNoPeersAvailable

}
func (pm *peerMap) Remove(pId peer.ID) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.m, pId)
}
func (pm *peerMap) Add(pId peer.ID) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.m[pId] = struct{}{}
}

type ServiceSlot struct {
	mu sync.Mutex
	m  map[protocol.ID]*peerMap
}

func NewServiceSlot() *ServiceSlot {
	return &ServiceSlot{
		m: map[protocol.ID]*peerMap{},
	}
}

func (slots *ServiceSlot) GetPeers(proto protocol.ID) *peerMap {
	slots.mu.Lock()
	defer slots.mu.Unlock()
	if slots.m[proto] == nil {
		slots.m[proto] = newPeerMap()
	}
	return slots.m[proto]
}
func (slots *ServiceSlot) RemovePeer(peerId peer.ID) {
	slots.mu.Lock()
	defer slots.mu.Unlock()
	for _, m := range slots.m {
		m.Remove(peerId)
	}
}
