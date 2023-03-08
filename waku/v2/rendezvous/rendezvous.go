package rendezvous

import (
	"context"
	"sync"

	r "github.com/berty/go-libp2p-rendezvous"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

const RendezvousID = r.RendezvousProto

type Rendezvous struct {
	host          host.Host
	peerConnector PeerConnector

	log *zap.Logger

	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

type PeerConnector interface {
	PeerChannel() chan<- peer.AddrInfo
}

func NewRendezvous(host host.Host, peerConnector PeerConnector, log *zap.Logger) *Rendezvous {
	logger := log.Named("rendezvous")

	return &Rendezvous{
		host:          host,
		peerConnector: peerConnector,
		wg:            &sync.WaitGroup{},
		log:           logger,
	}
}

func (r *Rendezvous) Start(ctx context.Context) error {
	r.wg.Wait() // Waiting for any go routines to stop
	ctx, cancel := context.WithCancel(ctx)

	r.cancel = cancel

	// Start service or goroutine that will push discovered nodes to peer connector?

	return nil
}

func (r *Rendezvous) Stop() {
	if r.cancel == nil {
		return
	}

	r.cancel()
	r.wg.Wait()
}
