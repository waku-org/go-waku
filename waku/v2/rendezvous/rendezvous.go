package rendezvous

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/backoff"
	rvs "github.com/waku-org/go-libp2p-rendezvous"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/peers"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"go.uber.org/zap"
)

const RendezvousID = rvs.RendezvousProto

type rendezvousPoint struct {
	sync.RWMutex

	id     peer.ID
	cookie []byte

	bkf     backoff.BackoffStrategy
	nextTry time.Time
}

type PeerConnector interface {
	Subscribe(context.Context, <-chan v2.PeerData)
}

type Rendezvous struct {
	host host.Host

	enableServer  bool
	db            *DB
	rendezvousSvc *rvs.RendezvousService

	rendezvousPoints []*rendezvousPoint
	peerConnector    PeerConnector

	log    *zap.Logger
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// NewRendezvous creates an instance of a Rendezvous which might act as rendezvous point for other nodes, or act as a client node
func NewRendezvous(enableServer bool, db *DB, rendezvousPoints []peer.ID, peerConnector PeerConnector, log *zap.Logger) *Rendezvous {
	logger := log.Named("rendezvous")

	rngSrc := rand.NewSource(rand.Int63())
	minBackoff, maxBackoff := time.Second*30, time.Hour
	bkf := backoff.NewExponentialBackoff(minBackoff, maxBackoff, backoff.FullJitter, time.Second, 5.0, 0, rand.New(rngSrc))

	var rendevousPoints []*rendezvousPoint
	now := time.Now()
	for _, rp := range rendezvousPoints {
		rendevousPoints = append(rendevousPoints, &rendezvousPoint{
			id:      rp,
			nextTry: now,
			bkf:     bkf(),
		})
	}

	return &Rendezvous{
		enableServer:     enableServer,
		db:               db,
		rendezvousPoints: rendevousPoints,
		peerConnector:    peerConnector,
		log:              logger,
	}
}

// Sets the host to be able to mount or consume a protocol
func (r *Rendezvous) SetHost(h host.Host) {
	r.host = h
}

func (r *Rendezvous) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	if r.enableServer {
		err := r.db.Start(ctx)
		if err != nil {
			cancel()
			return err
		}

		r.rendezvousSvc = rvs.NewRendezvousService(r.host, r.db)
	}

	r.log.Info("rendezvous protocol started")
	return nil
}

const registerBackoff = 200 * time.Millisecond
const registerMaxRetries = 7

func (r *Rendezvous) getRandomRendezvousPoint(ctx context.Context) <-chan *rendezvousPoint {
	var dialableRP []*rendezvousPoint
	now := time.Now()
	for _, rp := range r.rendezvousPoints {
		if now.After(rp.NextTry()) {
			dialableRP = append(dialableRP, rp)
		}
	}

	result := make(chan *rendezvousPoint, 1)

	if len(dialableRP) > 0 {
		result <- r.rendezvousPoints[rand.Intn(len(r.rendezvousPoints))] // nolint: gosec
	} else {
		if len(r.rendezvousPoints) > 0 {
			sort.Slice(r.rendezvousPoints, func(i, j int) bool {
				return r.rendezvousPoints[i].nextTry.Before(r.rendezvousPoints[j].nextTry)
			})

			tryIn := r.rendezvousPoints[0].NextTry().Sub(now)
			timer := time.NewTimer(tryIn)
			defer timer.Stop()

			select {
			case <-ctx.Done():
				break
			case <-timer.C:
				result <- r.rendezvousPoints[0]
			}
		}
	}

	close(result)
	return result
}

func (r *Rendezvous) Discover(ctx context.Context, topic string, numPeers int) {
	for {
		select {
		case <-ctx.Done():
			return
		case server, ok := <-r.getRandomRendezvousPoint(ctx):
			if !ok {
				return
			}

			rendezvousClient := rvs.NewRendezvousClient(r.host, server.id)

			addrInfo, cookie, err := rendezvousClient.Discover(ctx, topic, numPeers, server.cookie)
			if err != nil {
				r.log.Error("could not discover new peers", zap.Error(err))
				server.Delay()
				continue
			}

			if len(addrInfo) != 0 {
				server.SetSuccess(cookie)

				peerCh := make(chan v2.PeerData)
				r.peerConnector.Subscribe(context.Background(), peerCh)
				for _, addr := range addrInfo {
					peer := v2.PeerData{
						Origin:   peers.Rendezvous,
						AddrInfo: addr,
					}
					fmt.Println("PPPPPPPPPPPPPP")
					select {
					case peerCh <- peer:
						fmt.Println("DISCOVERED")
					case <-ctx.Done():
						return
					}
				}
				close(peerCh)
			} else {
				server.Delay()
			}
		}
	}
}

func (r *Rendezvous) DiscoverShard(ctx context.Context, cluster uint16, shard uint16, numPeers int) {
	namespace := ShardToNamespace(cluster, shard)
	r.Discover(ctx, namespace, numPeers)
}

func (r *Rendezvous) callRegister(ctx context.Context, rendezvousClient rvs.RendezvousClient, topic string, retries int) (<-chan time.Time, int) {
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

func (r *Rendezvous) Register(ctx context.Context, topic string) {
	for _, m := range r.rendezvousPoints {
		r.wg.Add(1)
		go func(m *rendezvousPoint) {
			r.wg.Done()

			rendezvousClient := rvs.NewRendezvousClient(r.host, m.id)
			retries := 0
			var t <-chan time.Time

			t, retries = r.callRegister(ctx, rendezvousClient, topic, retries)
			for {
				select {
				case <-ctx.Done():
					return
				case <-t:
					t, retries = r.callRegister(ctx, rendezvousClient, topic, retries)
					if retries >= registerMaxRetries {
						return
					}
				}
			}
		}(m)
	}
}

func (r *Rendezvous) RegisterShard(ctx context.Context, cluster uint16, shard uint16) {
	namespace := ShardToNamespace(cluster, shard)
	r.Register(ctx, namespace)
}

func (r *Rendezvous) RegisterRelayShards(ctx context.Context, rs protocol.RelayShards) {
	for _, idx := range rs.Indices {
		go r.RegisterShard(ctx, rs.Cluster, idx)
	}
}

func (r *Rendezvous) Stop() {
	r.cancel()
	r.wg.Wait()
	r.host.RemoveStreamHandler(rvs.RendezvousProto)
	r.rendezvousSvc = nil
}

func ShardToNamespace(cluster uint16, shard uint16) string {
	return fmt.Sprintf("rs/%d/%d", cluster, shard)
}

func (rp *rendezvousPoint) Delay() {
	rp.Lock()
	defer rp.Unlock()

	rp.nextTry = time.Now().Add(rp.bkf.Delay())
}

func (rp *rendezvousPoint) SetSuccess(cookie []byte) {
	rp.Lock()
	defer rp.Unlock()

	rp.bkf.Reset()
	rp.cookie = cookie
}

func (rp *rendezvousPoint) NextTry() time.Time {
	rp.RLock()
	defer rp.RUnlock()
	return rp.nextTry
}
