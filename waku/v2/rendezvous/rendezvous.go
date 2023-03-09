package rendezvous

import (
	"context"
	"math"
	"sync"
	"time"

	rvs "github.com/berty/go-libp2p-rendezvous"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
)

const RendezvousID = rvs.RendezvousProto

type Rendezvous struct {
	host host.Host

	enableServer  bool
	db            *DB
	rendezvousSvc *rvs.RendezvousService

	discoverPeers    bool
	rendezvousPoints []multiaddr.Multiaddr
	peerConnector    PeerConnector

	log    *zap.Logger
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

type PeerConnector interface {
	PeerChannel() chan<- peer.AddrInfo
}

func NewRendezvous(host host.Host, enableServer bool, db *DB, discoverPeers bool, rendevousPoints []multiaddr.Multiaddr, peerConnector PeerConnector, log *zap.Logger) *Rendezvous {
	logger := log.Named("rendezvous")

	return &Rendezvous{
		host:             host,
		enableServer:     enableServer,
		db:               db,
		discoverPeers:    discoverPeers,
		rendezvousPoints: rendevousPoints,
		peerConnector:    peerConnector,
		log:              logger,
	}
}

func (r *Rendezvous) Start(ctx context.Context) error {
	err := r.db.Start(ctx)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)

	r.cancel = cancel

	if r.enableServer {
		r.rendezvousSvc = rvs.NewRendezvousService(r.host, r.db)
	}

	if r.discoverPeers {
		r.wg.Add(1)
		go r.register(ctx)
	}

	// TODO: Execute discovery and push nodes to peer connector. If asking for peers fail, add timeout and exponential backoff

	r.log.Info("rendezvous protocol started")
	return nil
}

const registerBackoff = 200 * time.Millisecond
const registerMaxRetries = 7

func (r *Rendezvous) callRegister(ctx context.Context, rendezvousClient rvs.RendezvousClient, retries int) (<-chan time.Time, int) {
	ttl, err := rendezvousClient.Register(ctx, relay.DefaultWakuTopic, rvs.DefaultTTL) // TODO: determine which topic to use
	var t <-chan time.Time
	if err != nil {
		r.log.Error("registering rendezvous client", zap.Error(err))
		backoff := registerBackoff * time.Duration(math.Exp2(float64(retries)))
		t = time.After(backoff)
		retries++
	} else {
		t = time.After(ttl)
	}

	return t, retries
}

func (r *Rendezvous) register(ctx context.Context) {
	for _, m := range r.rendezvousPoints {
		r.wg.Add(1)
		go func(m multiaddr.Multiaddr) {
			r.wg.Done()

			peerIDStr, err := m.ValueForProtocol(multiaddr.P_P2P)
			if err != nil {
				r.log.Error("error obtaining peerID", zap.Error(err))
				return
			}

			peerID, err := peer.Decode(peerIDStr)
			if err != nil {
				r.log.Error("error obtaining peerID", zap.Error(err))
				return
			}

			rendezvousClient := rvs.NewRendezvousClient(r.host, peerID)
			retries := 0
			var t <-chan time.Time

			t, retries = r.callRegister(ctx, rendezvousClient, retries)
			for {
				select {
				case <-ctx.Done():
					return
				case <-t:
					t, retries = r.callRegister(ctx, rendezvousClient, retries)
					if retries >= registerMaxRetries {
						return
					}
				}
			}
		}(m)
	}
}

func (r *Rendezvous) Stop() {
	r.cancel()
	r.wg.Wait()
	r.host.RemoveStreamHandler(rvs.RendezvousProto)
	r.rendezvousSvc = nil
}
