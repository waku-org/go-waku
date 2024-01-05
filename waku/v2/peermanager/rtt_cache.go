package peermanager

import (
	"container/heap"
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/backoff"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"go.uber.org/zap"
)

type PeerRTT struct {
	pingQueueIndex int

	PeerID  peer.ID
	PingRTT time.Duration

	bkf              backoff.BackoffStrategy
	nextVerification time.Time
	timesource       timesource.Timesource
}

func (pr *PeerRTT) IsStale() bool {
	return pr.nextVerification.Before(pr.timesource.Now())
}

type RTTCache struct {
	sync.RWMutex

	host host.Host

	peers     map[peer.ID]*PeerRTT
	pingQueue PingQueue

	verificationInterval time.Duration

	timesource timesource.Timesource
	logger     *zap.Logger
}

func NewRTTCache(timesource timesource.Timesource, logger *zap.Logger) *RTTCache {
	return &RTTCache{
		timesource: timesource,
		logger:     logger.Named("ping-manager"),
		peers:      make(map[peer.ID]*PeerRTT),
	}
}

func (r *RTTCache) SetHost(h host.Host) {
	r.host = h
}

func (r *RTTCache) Start(ctx context.Context) {
	go r.start(ctx, 10*time.Second)
}

func (r *RTTCache) pingPeers(ctx context.Context) {
	r.RLock()
	var toPing peer.IDSlice
	if len(r.pingQueue) == 0 {
		toPing = r.host.Peerstore().Peers()
	} else {
		for _, p := range r.pingQueue {
			if p.IsStale() {
				toPing = append(toPing, p.PeerID)
			}
		}
	}
	r.RUnlock()
	for _, p := range toPing {
		go r.PingPeer(ctx, p)
	}
}

func (r *RTTCache) start(ctx context.Context, pingInterval time.Duration) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.pingPeers(ctx)
		}
	}

}

func (r *RTTCache) PingPeer(ctx context.Context, peer peer.ID) (*PeerRTT, error) {
	if peer == r.host.ID() {
		return nil, nil // Don't ping yourself
	}

	r.RLock()
	peerRTT, ok := r.peers[peer]
	r.RUnlock()

	if ok && !peerRTT.IsStale() {
		// We already have a ping recorded
		return peerRTT, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case result := <-ping.Ping(ctx, r.host, peer):
		r.Lock()
		defer r.Unlock()

		peerRTT, exists := r.getOrDefault(peer)
		if result.Error == nil {
			peerRTT.PingRTT = result.RTT
			peerRTT.nextVerification = time.Now().Add(r.verificationInterval)
			if exists {
				r.pingQueue.Update(peerRTT)
			} else {
				r.pingQueue.Push(peerRTT)
			}
			return peerRTT, nil
		} else {
			peerRTT.PingRTT = time.Hour // This is just so we don't choose this peer again since the RTT is unreasonable
			peerRTT.nextVerification = time.Now().Add(peerRTT.bkf.Delay())
			if exists {
				r.pingQueue.Update(peerRTT)
			} else {
				r.pingQueue.Push(peerRTT)
			}

			r.logger.Debug("could not ping", logging.HostID("peer", peer), zap.Error(result.Error))

			return nil, result.Error
		}
	}
}

func (r *RTTCache) PeersRTT(ctx context.Context, peers peer.IDSlice) []*PeerRTT {
	wg := sync.WaitGroup{}
	pingCh := make(chan *PeerRTT, len(peers))
	wg.Add(len(peers))
	for _, p := range peers {
		go func(p peer.ID) {
			defer wg.Done()
			peerRTT, err := r.PingPeer(ctx, p)
			if err == nil {
				pingCh <- peerRTT
			}
		}(p)
	}

	wg.Wait()
	close(pingCh)

	var result []*PeerRTT
	for peerRTT := range pingCh {
		result = append(result, peerRTT)
	}

	return result
}

func (r *RTTCache) FastestPeer(ctx context.Context, peers peer.IDSlice) (*PeerRTT, error) {
	var min *PeerRTT
	for _, p := range r.PeersRTT(ctx, peers) {
		if min == nil {
			min = p
		} else {
			if p.PingRTT < min.PingRTT {
				min = p
			}
		}
	}
	if min == nil || min.PingRTT == time.Hour {
		return nil, ErrNoPeersAvailable
	}

	return min, nil
}

func (r *RTTCache) getOrDefault(peerID peer.ID) (*PeerRTT, bool) {
	peerRTT, ok := r.peers[peerID]
	exists := true

	if !ok {

		rngSrc := rand.NewSource(rand.Int63())
		minBackoff, maxBackoff := time.Second*5, time.Hour
		bkf := backoff.NewExponentialBackoff(minBackoff, maxBackoff, backoff.FullJitter, time.Second, 5.0, 0, rand.New(rngSrc))()

		peerRTT = &PeerRTT{
			PeerID:           peerID,
			nextVerification: r.timesource.Now().Add(bkf.Delay()),
			bkf:              bkf,
			timesource:       r.timesource,
		}

		r.peers[peerID] = peerRTT

		exists = false
	}

	return peerRTT, exists
}

type PingQueue []*PeerRTT

func (pq PingQueue) Len() int { return len(pq) }

func (pq PingQueue) Less(i, j int) bool {
	if pq[i].nextVerification.IsZero() || pq[j].nextVerification.IsZero() {
		return false
	}

	return pq[i].nextVerification.After(pq[j].nextVerification)
}

func (pq PingQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].pingQueueIndex = i
	pq[j].pingQueueIndex = j
}

func (pq *PingQueue) Push(x any) {
	n := len(*pq)
	item := x.(*PeerRTT)
	item.pingQueueIndex = n
	*pq = append(*pq, item)
}

func (pq *PingQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil           // avoid memory leak
	item.pingQueueIndex = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *PingQueue) Remove(item *PeerRTT) {
	heap.Remove(pq, item.pingQueueIndex)
}

func (pq *PingQueue) Update(newItem *PeerRTT) {
	heap.Fix(pq, newItem.pingQueueIndex)
}
