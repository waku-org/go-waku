package rendezvous

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	rvs "github.com/waku-org/go-libp2p-rendezvous"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/peers"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"go.uber.org/zap"
)

const RendezvousID = rvs.RendezvousProto
const RegisterDefaultTTL = rvs.DefaultTTL * time.Second

type Rendezvous struct {
	host host.Host

	db            *DB
	rendezvousSvc *rvs.RendezvousService

	peerConnector PeerConnector

	log    *zap.Logger
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

type PeerConnector interface {
	Subscribe(context.Context, <-chan v2.PeerData)
}

func NewRendezvous(db *DB, peerConnector PeerConnector, log *zap.Logger) *Rendezvous {
	logger := log.Named("rendezvous")
	return &Rendezvous{
		db:            db,
		peerConnector: peerConnector,
		log:           logger,
	}
}

// Sets the host to be able to mount or consume a protocol
func (r *Rendezvous) SetHost(h host.Host) {
	r.host = h
}

func (r *Rendezvous) Start(ctx context.Context) error {
	if r.cancel != nil {
		return errors.New("already started")
	}

	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	err := r.db.Start(ctx)
	if err != nil {
		cancel()
		return err
	}

	r.rendezvousSvc = rvs.NewRendezvousService(r.host, r.db)

	r.log.Info("rendezvous protocol started")
	return nil
}

const registerBackoff = 200 * time.Millisecond
const registerMaxRetries = 7

func (r *Rendezvous) Discover(ctx context.Context, rp *RendezvousPoint, numPeers int) {
	r.DiscoverWithTopic(ctx, protocol.DefaultPubsubTopic().String(), rp, numPeers)
}

func (r *Rendezvous) DiscoverShard(ctx context.Context, rp *RendezvousPoint, cluster uint16, shard uint16, numPeers int) {
	namespace := ShardToNamespace(cluster, shard)
	r.DiscoverWithTopic(ctx, namespace, rp, numPeers)
}

func (r *Rendezvous) DiscoverWithTopic(ctx context.Context, topic string, rp *RendezvousPoint, numPeers int) {
	rendezvousClient := rvs.NewRendezvousClient(r.host, rp.id)

	addrInfo, cookie, err := rendezvousClient.Discover(ctx, topic, numPeers, rp.cookie)
	if err != nil {
		r.log.Error("could not discover new peers", zap.Error(err))
		rp.Delay()
		return
	}

	if len(addrInfo) != 0 {
		rp.SetSuccess(cookie)

		peerCh := make(chan v2.PeerData)
		defer close(peerCh)
		r.peerConnector.Subscribe(ctx, peerCh)
		for _, p := range addrInfo {
			peer := v2.PeerData{
				Origin:   peers.Rendezvous,
				AddrInfo: p,
			}
			select {
			case <-ctx.Done():
				return
			case peerCh <- peer:
			}
		}
	} else {
		rp.Delay()
	}

}

func (r *Rendezvous) callRegister(ctx context.Context, topic string, rendezvousClient rvs.RendezvousClient, retries int) (<-chan time.Time, int) {
	ttl, err := rendezvousClient.Register(ctx, topic, rvs.DefaultTTL)
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

func (r *Rendezvous) Register(ctx context.Context, rendezvousPoints []*RendezvousPoint) {
	r.RegisterWithTopic(ctx, protocol.DefaultPubsubTopic().String(), rendezvousPoints)
}

func (r *Rendezvous) RegisterShard(ctx context.Context, cluster uint16, shard uint16, rendezvousPoints []*RendezvousPoint) {
	namespace := ShardToNamespace(cluster, shard)
	r.RegisterWithTopic(ctx, namespace, rendezvousPoints)
}

func (r *Rendezvous) RegisterRelayShards(ctx context.Context, rs protocol.RelayShards, rendezvousPoints []*RendezvousPoint) {
	for _, idx := range rs.Indices {
		go r.RegisterShard(ctx, rs.Cluster, idx, rendezvousPoints)
	}
}

func (r *Rendezvous) RegisterWithTopic(ctx context.Context, topic string, rendezvousPoints []*RendezvousPoint) {
	for _, m := range rendezvousPoints {
		r.wg.Add(1)
		go func(m *RendezvousPoint) {
			r.wg.Done()

			rendezvousClient := rvs.NewRendezvousClient(r.host, m.id)
			retries := 0
			var t <-chan time.Time

			t, retries = r.callRegister(ctx, topic, rendezvousClient, retries)
			for {
				select {
				case <-ctx.Done():
					return
				case <-t:
					t, retries = r.callRegister(ctx, topic, rendezvousClient, retries)
					if retries >= registerMaxRetries {
						return
					}
				}
			}
		}(m)
	}
}

func (r *Rendezvous) Stop() {
	if r.cancel == nil {
		return
	}

	r.cancel()
	r.wg.Wait()
	r.host.RemoveStreamHandler(rvs.RendezvousProto)
	r.rendezvousSvc = nil
}

func ShardToNamespace(cluster uint16, shard uint16) string {
	return fmt.Sprintf("rs/%d/%d", cluster, shard)
}
