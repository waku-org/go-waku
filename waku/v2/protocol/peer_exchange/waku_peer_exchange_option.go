package peer_exchange

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"go.uber.org/zap"
)

type PeerExchangeParameters struct {
	host         host.Host
	selectedPeer peer.ID
	pm           *peermanager.PeerManager
	log          *zap.Logger
}

type PeerExchangeOption func(*PeerExchangeParameters)

// WithPeer is an option used to specify the peerID to push a waku message to
func WithPeer(p peer.ID) PeerExchangeOption {
	return func(params *PeerExchangeParameters) {
		params.selectedPeer = p
	}
}

// WithAutomaticPeerSelection is an option used to randomly select a peer from the Waku peer store
// to obtains peers from. If a list of specific peers is passed, the peer will be chosen
// from that list assuming it supports the chosen protocol, otherwise it will chose a peer
// from the node peerstore
// Note: this option can only be used if WakuNode is initialized which internally intializes the peerManager
func WithAutomaticPeerSelection(fromThesePeers ...peer.ID) PeerExchangeOption {
	return func(params *PeerExchangeParameters) {
		if params.pm == nil {
			params.log.Info("automatic selection is not avaiable since peerManager is not initialized")
		} else {
			p, err := params.pm.SelectPeer(peermanager.PeerSelectionCriteria{Proto: PeerExchangeID_v20alpha1,
				SpecificPeers: fromThesePeers})
			if err == nil {
				params.selectedPeer = p
			} else {
				params.log.Info("selecting peer", zap.Error(err))
			}
		}
	}
}

// WithFastestPeerSelection is an option used to select a peer from the peer store
// with the lowest ping. If a list of specific peers is passed, the peer will be chosen
// from that list assuming it supports the chosen protocol, otherwise it will chose a peer
// from the node peerstore
func WithFastestPeerSelection(ctx context.Context, fromThesePeers ...peer.ID) PeerExchangeOption {
	return func(params *PeerExchangeParameters) {
		if params.pm == nil {
			params.log.Info("automatic selection is not avaiable since peerManager is not initialized")
		} else {
			p, err := params.pm.SelectPeerWithLowestRTT(
				peermanager.PeerSelectionCriteria{Proto: PeerExchangeID_v20alpha1,
					SpecificPeers: fromThesePeers, Ctx: ctx})
			if err == nil {
				params.selectedPeer = p
			} else {
				params.log.Info("selecting peer", zap.Error(err))
			}
		}
	}
}

// DefaultOptions are the default options to be used when using the lightpush protocol
func DefaultOptions(host host.Host) []PeerExchangeOption {
	return []PeerExchangeOption{
		WithAutomaticPeerSelection(),
	}
}
