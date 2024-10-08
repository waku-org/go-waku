package service

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/peer"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

// PeerData contains information about a peer useful in establishing connections with it.
type PeerData struct {
	Origin       wps.Origin
	AddrInfo     peer.AddrInfo
	ENR          *enode.Node
	PubsubTopics []string
}

type CommonDiscoveryService struct {
	commonService *CommonService

	channel         chan PeerData
	channelIsClosed bool
	channelMutex    sync.Mutex
}

func NewCommonDiscoveryService() *CommonDiscoveryService {
	return &CommonDiscoveryService{
		commonService: NewCommonService(),
	}
}

func (sp *CommonDiscoveryService) Start(ctx context.Context, fn func() error) error {
	return sp.commonService.Start(ctx, func() error {
		// currently is used in discv5,peerConnector,rendevzous for returning new discovered Peers to peerConnector for connecting with them
		// mutex protection for this operation
		sp.channelMutex.Lock()
		sp.channel = make(chan PeerData)
		sp.channelIsClosed = false
		sp.channelMutex.Unlock()
		return fn()
	})
}

func (sp *CommonDiscoveryService) Stop(stopFn func()) {
	sp.commonService.Stop(func() {
		stopFn()
		sp.WaitGroup().Wait() // waitgroup is waited here so that channel can be closed after all the go rountines have stopped in service.
		// there is a wait in the CommonService too
		sp.channelMutex.Lock()
		close(sp.channel)
		sp.channelIsClosed = true
		sp.channelMutex.Unlock()
	})
}
func (sp *CommonDiscoveryService) GetListeningChan() <-chan PeerData {
	return sp.channel
}
func (sp *CommonDiscoveryService) PushToChan(data PeerData) bool {
	defer utils.LogOnPanic()
	if err := sp.ErrOnNotRunning(); err != nil {
		return false
	}

	sp.channelMutex.Lock()
	defer sp.channelMutex.Unlock()

	if sp.channelIsClosed {
		return false
	}

	select {
	case sp.channel <- data:
		return true
	case <-sp.Context().Done():
		return false
	}
}

func (sp *CommonDiscoveryService) RLock() {
	sp.commonService.RLock()
}
func (sp *CommonDiscoveryService) RUnlock() {
	sp.commonService.RUnlock()
}

func (sp *CommonDiscoveryService) Context() context.Context {
	return sp.commonService.Context()
}
func (sp *CommonDiscoveryService) ErrOnNotRunning() error {
	return sp.commonService.ErrOnNotRunning()
}
func (sp *CommonDiscoveryService) WaitGroup() *sync.WaitGroup {
	return sp.commonService.WaitGroup()
}
