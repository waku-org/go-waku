package peermanager

import (
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
	InRelayPeersTarget  uint
	OutRelayPeersTarget uint
	host                host.Host
}

const maxRelayPeersShare = 5
const defaultMaxOutRelayPeersTarget = 10
const outRelayPeersShare = 3
const peerConnectivityLoopSecs = 15

// NewPeerManager creates a new peerManager instance.
func NewPeerManager(maxConnections uint, logger *zap.Logger) *PeerManager {

	maxRelayPeersValue := maxConnections - (maxConnections / maxRelayPeersShare)
	outRelayPeersTargetValue := uint(maxRelayPeersValue / outRelayPeersShare)

	pm := &PeerManager{
		logger:              logger.Named("peer-manager"),
		maxRelayPeers:       maxRelayPeersValue,
		InRelayPeersTarget:  maxRelayPeersValue - outRelayPeersTargetValue,
		OutRelayPeersTarget: outRelayPeersTargetValue,
	}
	logger.Info("PeerManager init values", zap.Uint("maxConnections", maxConnections),
		zap.Uint("maxRelayPeersValue", maxRelayPeersValue), zap.Uint("outRelayPeersTargetValue", outRelayPeersTargetValue),
		zap.Uint("inRelayPeersTarget", pm.InRelayPeersTarget))

	return pm
}

func (pm *PeerManager) SetHost(host host.Host) {
	pm.host = host
}

// Start starts the processing to be done by peer manager.
func (pm *PeerManager) Start() {
	go pm.connectivityLoop()
}

// This is a connectivity loop, which currently checks and prunes inbound connections.
func (pm *PeerManager) connectivityLoop() {

	for {
		pm.pruneInRelayConns()
		time.Sleep(peerConnectivityLoopSecs * time.Second)
	}
}

func (pm *PeerManager) filterPeersByProto(peers peer.IDSlice, proto ...protocol.ID) peer.IDSlice {
	var filteredPeers peer.IDSlice
	//TODO: This can be optimized once we have waku's own peerStore

	for _, p := range peers {
		supportedProtocols, err := pm.host.Peerstore().SupportsProtocols(p, proto...)
		if err != nil {
			pm.logger.Warn("Failed to get supported protocols for peer", zap.String("peerID", p.String()))
		        continue
		}
		if len(supportedProtocols) != 0 {
			filteredPeers = append(filteredPeers, p)
		}
	}
	return filteredPeers
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
	inRelayPeers = pm.filterPeersByProto(inPeers, WakuRelayIDv200)
	outRelayPeers := pm.filterPeersByProto(outPeers, WakuRelayIDv200)
	pm.logger.Info("Number of Relay peers connected", zap.Int("inRelayPeers", inRelayPeers.Len()), zap.Int("outRelayPeers", outRelayPeers.Len()))

	if inRelayPeers.Len() > int(pm.InRelayPeersTarget) {
		//Start disconnecting peers, based on what?
		//For now, just disconnect most recently connected peers
		//TODO: Need to have more intelligent way of doing this, maybe peer scores.
		pm.logger.Info("Number of in peer connections exceed targer relay peers, hence pruning", zap.Int("inRelayPeers", inRelayPeers.Len()), zap.Uint("inRelayPeersTarget", pm.InRelayPeersTarget))
		for pruningStartIndex := pm.InRelayPeersTarget; pruningStartIndex < uint(inRelayPeers.Len()); pruningStartIndex++ {
			p := inRelayPeers[pruningStartIndex]
			err := pm.host.Network().ClosePeer(p)
			if err != nil {
				pm.logger.Warn("Failed to disconnect connection towards peer", zap.String("peerID", p.String()))
			}
			pm.host.Peerstore().RemovePeer(p) //TODO: Should we remove the peer immediately?
			err = pm.host.Peerstore().(wps.WakuPeerstore).SetDirection(p, network.DirUnknown)
			if err != nil {
				pm.logger.Warn("Failed to remove metadata for peer", zap.String("peerID", p.String()))
			}
			pm.logger.Info("Successfully disconnected connection towards peer", zap.String("peerID", p.String()))
		}
	}
}
