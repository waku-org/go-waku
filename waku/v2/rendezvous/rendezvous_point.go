package rendezvous

import (
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/backoff"
)

type RendezvousPoint struct {
	sync.RWMutex

	id     peer.ID
	cookie []byte

	bkf     backoff.BackoffStrategy
	nextTry time.Time
}

func NewRendezvousPoint(peerID peer.ID) *RendezvousPoint {
	rngSrc := rand.NewSource(rand.Int63())
	minBackoff, maxBackoff := time.Second*30, time.Hour
	bkf := backoff.NewExponentialBackoff(minBackoff, maxBackoff, backoff.FullJitter, time.Second, 5.0, 0, rand.New(rngSrc))

	now := time.Now()

	rp := &RendezvousPoint{
		id:      peerID,
		nextTry: now,
		bkf:     bkf(),
	}

	return rp
}

func (rp *RendezvousPoint) Delay() {
	rp.Lock()
	defer rp.Unlock()

	rp.nextTry = time.Now().Add(rp.bkf.Delay())
}

func (rp *RendezvousPoint) SetSuccess(cookie []byte) {
	rp.Lock()
	defer rp.Unlock()

	rp.bkf.Reset()
	rp.cookie = cookie
}

func (rp *RendezvousPoint) NextTry() time.Time {
	rp.RLock()
	defer rp.RUnlock()
	return rp.nextTry
}
