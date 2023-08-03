package peermanager

import (
	"math"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"golang.org/x/exp/slices"

	"go.uber.org/zap"
)

// TODO: Move all the protocol IDs to a common location.
// WakuRelayIDv200 is protocol ID for Waku v2 relay protocol
const WakuRelayIDv200 = protocol.ID("/vac/waku/relay/2.0.0")

// PeerManager applies various controls and manage connections towards peers.
type PeerManager struct {
	maxRelayPeers       uint
	logger              *zap.Logger
	inRelayPeersTarget  uint
	outRelayPeersTarget uint
	host                host.Host
}

const maxRelayPeersShare = 5
const defaultMaxOutRelayPeersTarget = 10
const outRelayPeersShare = 3

// NewPeerManager creates a new peerManager instance.
func NewPeerManager(maxConnections uint, host host.Host, logger *zap.Logger) *PeerManager {
	maxRelayPeersValue := maxConnections - (maxConnections / maxRelayPeersShare)
	var outRelayPeersTargetValue uint
	if maxRelayPeersValue > defaultMaxOutRelayPeersTarget {
		outRelayPeersTargetValue = uint(math.Max(float64(maxRelayPeersValue/outRelayPeersShare), defaultMaxOutRelayPeersTarget))
	} else {
		outRelayPeersTargetValue = maxRelayPeersValue / outRelayPeersShare
	}
	pm := &PeerManager{
		logger:              logger.Named("peer-manager"),
		maxRelayPeers:       maxRelayPeersValue,
		inRelayPeersTarget:  maxRelayPeersValue - outRelayPeersTargetValue,
		outRelayPeersTarget: outRelayPeersTargetValue,
		host:                host,
	}
	logger.Info("PeerManager init values", zap.Uint("maxConnections", maxConnections),
		zap.Uint("maxRelayPeersValue", maxRelayPeersValue), zap.Uint("outRelayPeersTargetValue", outRelayPeersTargetValue),
		zap.Uint("inRelayPeersTarget", pm.inRelayPeersTarget))

	return pm
}

// Start starts the processing to be done by peer manager.
func (pm *PeerManager) Start() {
	go pm.connectivityLoop()
}

// This is a connectivity loop, which currently checks and prunes inbound connections.
func (pm *PeerManager) connectivityLoop() {

	for {
		pm.pruneInRelayConns()
		time.Sleep(5 * time.Second)
	}
}

func (pm *PeerManager) pruneInRelayConns() {

	var inRelayPeers peer.IDSlice
	//Group peers by their connected direction inbound or outbound.
	inPeers, outPeers, err := pm.host.Peerstore().(wps.WakuPeerstore).GroupPeersByDirection()
	if err != nil {
		return
	}
	pm.logger.Info("Number of peers connected", zap.Int("inPeers", inPeers.Len()), zap.Int("outPeers", outPeers.Len()))

	//Need to filter peers to check if they support relay
	//TODO: This can be optimized once we have waku's own peerStore

	for _, p := range inPeers {
		supportedProtocols, err := pm.host.Peerstore().GetProtocols(p)
		if err != nil {
			pm.logger.Warn("Failed to get supported protocols for peer", zap.String("peerID", p.String()))
		}
		if slices.Contains(supportedProtocols, WakuRelayIDv200) {
			inRelayPeers = append(inRelayPeers, p)
		}
	}

	if inRelayPeers.Len() > int(pm.inRelayPeersTarget) {
		//Start disconnecting peers, based on what?
		//For now, just disconnect most recently connected peers
		//TODO: Need to have more intelligent way of doing this, maybe peer scores.
		pm.logger.Info("Number of in peer connections exceed targer relay peers, hence pruning", zap.Int("inRelayPeers", inRelayPeers.Len()), zap.Uint("inRelayPeersTarget", pm.inRelayPeersTarget))
		for pruningStartIndex := pm.inRelayPeersTarget; pruningStartIndex < uint(inRelayPeers.Len()); pruningStartIndex++ {
			p := inRelayPeers[pruningStartIndex]
			err := pm.host.Network().ClosePeer(p)
			if err != nil {
				pm.logger.Warn("Failed to disconnect connection towards peer", zap.String("peerID", p.String()))
			}
			pm.host.Peerstore().RemovePeer(p) //TODO: Should we remove the peer immediately?
			pm.host.Peerstore().(wps.WakuPeerstore).SetDirection(p, network.DirUnknown)
			pm.logger.Info("Successfully disconnected connection towards peer", zap.String("peerID", p.String()))
		}
	}
}
