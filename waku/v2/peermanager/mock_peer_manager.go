package peermanager

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

type TestPeerDiscoverer struct {
	sync.RWMutex
	peerMap map[peer.ID]struct{}
}

func NewTestPeerDiscoverer() *TestPeerDiscoverer {
	result := &TestPeerDiscoverer{
		peerMap: make(map[peer.ID]struct{}),
	}

	return result
}

func (t *TestPeerDiscoverer) Subscribe(ctx context.Context, ch <-chan PeerData) {
	go func() {
		for p := range ch {
			t.Lock()
			t.peerMap[p.AddrInfo.ID] = struct{}{}
			t.Unlock()
		}
	}()
}

func (t *TestPeerDiscoverer) HasPeer(p peer.ID) bool {
	t.RLock()
	defer t.RUnlock()
	_, ok := t.peerMap[p]
	return ok
}

func (t *TestPeerDiscoverer) PeerCount() int {
	t.RLock()
	defer t.RUnlock()
	return len(t.peerMap)
}

func (t *TestPeerDiscoverer) Clear() {
	t.Lock()
	defer t.Unlock()
	t.peerMap = make(map[peer.ID]struct{})
}
