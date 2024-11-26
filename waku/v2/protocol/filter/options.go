package filter

import (
	"errors"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func (old *SubscribeParameters) Copy() *SubscribeParameters {
	return &SubscribeParameters{
		selectedPeers: old.selectedPeers,
		requestID:     old.requestID,
	}
}

type (
	PingParameters struct {
		requestID []byte
	}
	PingOption func(*PingParameters)
)

func WithPingRequestId(requestId []byte) PingOption {
	return func(params *PingParameters) {
		params.requestID = requestId
	}
}

type (
	SubscribeParameters struct {
		selectedPeers     peer.IDSlice
		peerAddr          multiaddr.Multiaddr
		peerSelectionType peermanager.PeerSelection
		preferredPeers    peer.IDSlice
		peersToExclude    peermanager.PeerSet
		maxPeers          int
		requestID         []byte
		log               *zap.Logger

		// Subscribe-specific
		host host.Host
		pm   *peermanager.PeerManager

		// Unsubscribe-specific
		unsubscribeAll bool
		wg             *sync.WaitGroup
	}

	FullNodeParameters struct {
		Timeout        time.Duration
		MaxSubscribers int
		pm             *peermanager.PeerManager
		limitR         rate.Limit
		limitB         int
	}

	FullNodeOption func(*FullNodeParameters)

	LightNodeParameters struct {
		limitR rate.Limit
		limitB int
	}

	LightNodeOption func(*LightNodeParameters)

	SubscribeOption func(*SubscribeParameters) error
)

func WithLightNodeRateLimiter(r rate.Limit, b int) LightNodeOption {
	return func(params *LightNodeParameters) {
		params.limitR = r
		params.limitB = b
	}
}

func DefaultLightNodeOptions() []LightNodeOption {
	return []LightNodeOption{
		WithLightNodeRateLimiter(rate.Inf, 0),
	}
}

func WithTimeout(timeout time.Duration) FullNodeOption {
	return func(params *FullNodeParameters) {
		params.Timeout = timeout
	}
}

// WithPeer is an option used to specify the peerID to request the message history.
// Note that this option is mutually exclusive to WithPeerAddr, only one of them can be used.
func WithPeer(p peer.ID) SubscribeOption {
	return func(params *SubscribeParameters) error {
		params.selectedPeers = append(params.selectedPeers, p)
		if params.peerAddr != nil {
			return errors.New("peerAddr and peerId options are mutually exclusive")
		}
		return nil
	}
}

// WithPeerAddr is an option used to specify a peerAddress.
// This new peer will be added to peerStore.
// Note that this option is mutually exclusive to WithPeerAddr, only one of them can be used.
func WithPeerAddr(pAddr multiaddr.Multiaddr) SubscribeOption {
	return func(params *SubscribeParameters) error {
		params.peerAddr = pAddr
		if len(params.selectedPeers) != 0 {
			return errors.New("peerAddr and peerId options are mutually exclusive")
		}
		return nil
	}
}

func WithMaxPeersPerContentFilter(numPeers int) SubscribeOption {
	return func(params *SubscribeParameters) error {
		params.maxPeers = numPeers
		return nil
	}
}

// WithPeersToExclude option excludes the peers that are specified from selection
func WithPeersToExclude(peers ...peer.ID) SubscribeOption {
	return func(params *SubscribeParameters) error {
		params.peersToExclude = peermanager.PeerSliceToMap(peers)
		return nil
	}
}

// WithAutomaticPeerSelection is an option used to randomly select a peer from the peer store.
// If a list of specific peers is passed, the peer will be chosen from that list assuming it
// supports the chosen protocol, otherwise it will chose a peer from the node peerstore
func WithAutomaticPeerSelection(fromThesePeers ...peer.ID) SubscribeOption {
	return func(params *SubscribeParameters) error {
		params.peerSelectionType = peermanager.Automatic
		params.preferredPeers = fromThesePeers
		return nil
	}
}

// WithFastestPeerSelection is an option used to select a peer from the peer store
// with the lowest ping If a list of specific peers is passed, the peer will be chosen
// from that list assuming it supports the chosen protocol, otherwise it will chose a
// peer from the node peerstore
func WithFastestPeerSelection(fromThesePeers ...peer.ID) SubscribeOption {
	return func(params *SubscribeParameters) error {
		params.peerSelectionType = peermanager.LowestRTT
		return nil
	}
}

// WithRequestID is an option to set a specific request ID to be used when
// creating/removing a filter subscription
func WithRequestID(requestID []byte) SubscribeOption {
	return func(params *SubscribeParameters) error {
		params.requestID = requestID
		return nil
	}
}

// WithAutomaticRequestID is an option to automatically generate a request ID
// when creating a filter subscription
func WithAutomaticRequestID() SubscribeOption {
	return func(params *SubscribeParameters) error {
		params.requestID = protocol.GenerateRequestID()
		return nil
	}
}

func DefaultSubscriptionOptions() []SubscribeOption {
	return []SubscribeOption{
		WithAutomaticPeerSelection(),
		WithAutomaticRequestID(),
		WithMaxPeersPerContentFilter(1),
	}
}

func UnsubscribeAll() SubscribeOption {
	return func(params *SubscribeParameters) error {
		params.unsubscribeAll = true
		return nil
	}
}

// WithWaitGroup allows specifying a waitgroup to wait until all
// unsubscribe requests are complete before the function is complete
func WithWaitGroup(wg *sync.WaitGroup) SubscribeOption {
	return func(params *SubscribeParameters) error {
		params.wg = wg
		return nil
	}
}

// DontWait is used to fire and forget an unsubscription, and don't
// care about the results of it
func DontWait() SubscribeOption {
	return func(params *SubscribeParameters) error {
		params.wg = nil
		return nil
	}
}

func DefaultUnsubscribeOptions() []SubscribeOption {
	return []SubscribeOption{
		WithAutomaticRequestID(),
		WithWaitGroup(&sync.WaitGroup{}),
	}
}

func WithMaxSubscribers(maxSubscribers int) FullNodeOption {
	return func(params *FullNodeParameters) {
		params.MaxSubscribers = maxSubscribers
	}
}

func WithPeerManager(pm *peermanager.PeerManager) FullNodeOption {
	return func(params *FullNodeParameters) {
		params.pm = pm
	}
}

func WithFullNodeRateLimiter(r rate.Limit, b int) FullNodeOption {
	return func(params *FullNodeParameters) {
		params.limitR = r
		params.limitB = b
	}
}

func DefaultFullNodeOptions() []FullNodeOption {
	return []FullNodeOption{
		WithTimeout(DefaultIdleSubscriptionTimeout),
		WithMaxSubscribers(DefaultMaxSubscribers),
		WithFullNodeRateLimiter(rate.Inf, 0),
	}
}
