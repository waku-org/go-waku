package lightpush

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

type lightPushParameters struct {
	host              host.Host
	selectedPeer      peer.ID
	peerSelectionType utils.PeerSelection
	preferredPeers    peer.IDSlice
	requestID         []byte
	pm                *peermanager.PeerManager
	log               *zap.Logger
	pubsubTopic       string
}

// Option is the type of options accepted when performing LightPush protocol requests
type Option func(*lightPushParameters)

// WithPeer is an option used to specify the peerID to push a waku message to
func WithPeer(p peer.ID) Option {
	return func(params *lightPushParameters) {
		params.selectedPeer = p
	}
}

// WithAutomaticPeerSelection is an option used to randomly select a peer from the peer store
// to push a waku message to. If a list of specific peers is passed, the peer will be chosen
// from that list assuming it supports the chosen protocol, otherwise it will chose a peer
// from the node peerstore
func WithAutomaticPeerSelection(fromThesePeers ...peer.ID) Option {
	return func(params *lightPushParameters) {
		params.peerSelectionType = utils.Automatic
		params.preferredPeers = fromThesePeers
	}
}

func WithPubSubTopic(pubsubTopic string) Option {
	return func(params *lightPushParameters) {
		params.pubsubTopic = pubsubTopic
	}
}

// WithFastestPeerSelection is an option used to select a peer from the peer store
// with the lowest ping. If a list of specific peers is passed, the peer will be chosen
// from that list assuming it supports the chosen protocol, otherwise it will chose a peer
// from the node peerstore
func WithFastestPeerSelection(ctx context.Context, fromThesePeers ...peer.ID) Option {
	return func(params *lightPushParameters) {
		params.peerSelectionType = utils.LowestRTT
	}
}

// WithRequestID is an option to set a specific request ID to be used when
// publishing a message
func WithRequestID(requestID []byte) Option {
	return func(params *lightPushParameters) {
		params.requestID = requestID
	}
}

// WithAutomaticRequestID is an option to automatically generate a request ID
// when publishing a message
func WithAutomaticRequestID() Option {
	return func(params *lightPushParameters) {
		params.requestID = protocol.GenerateRequestID()
	}
}

// DefaultOptions are the default options to be used when using the lightpush protocol
func DefaultOptions(host host.Host) []Option {
	return []Option{
		WithAutomaticRequestID(),
		WithAutomaticPeerSelection(),
	}
}
