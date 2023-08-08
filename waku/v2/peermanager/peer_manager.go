package peermanager

import (
	"context"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/utils"

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
	serviceSlots        map[string]peer.ID
}

const maxRelayPeersShare = 5

// const defaultMaxOutRelayPeersTarget = 10
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
		serviceSlots:        make(map[string]peer.ID),
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
func (pm *PeerManager) Start(ctx context.Context) {
	go pm.connectivityLoop(ctx)
}

// This is a connectivity loop, which currently checks and prunes inbound connections.
func (pm *PeerManager) connectivityLoop(ctx context.Context) {
	t := time.NewTicker(peerConnectivityLoopSecs * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pm.pruneInRelayConns()
		}
	}
}

// filterPeersByProto filters peers from the peerSet based on protocols passed.
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
			pm.logger.Info("Successfully disconnected connection towards peer", zap.String("peerID", p.String()))
		}
	}
}

// AddPeer adds peer to the peerStore
func (pm *PeerManager) AddPeer(peerID peer.ID, addrs []multiaddr.Multiaddr, enr *enode.Node, origin wps.Origin) {
	// TODO: Move all peer addition logic to peermanager.
	//TODO: Add peer to peerStore
	//Set origin, ENR and Direction and any other required items in Waku Peer Store.

}

// RemovePeer deletes peer from the peerStore after disconnecting it.
func (pm *PeerManager) RemovePeer(peerID peer.ID) {
	// TODO: Need to handle removePeer also via peermanager
	// Need to updated serviceSlot accordingly.

}

// AddServicePeer adds a peerID to serviceSlot of peerManager.
// Adding to peerStore is already done by caller.
// If relay proto is passed, it is not added to serviceSlot.
// For relay peers use AddPeer.
func (pm *PeerManager) AddServicePeer(proto string, peerID peer.ID) {
	if proto == string(WakuRelayIDv200) {
		pm.logger.Warn("Cannot add Relay peer to service peer slots")
		return
	}
	//For now adding the peer to serviceSlot which means the latest added peer would be given priority.
	//TODO: Ideally we should maintain multiple peers per service and return best peer based on peer score or RTT etc.
	pm.logger.Info("Adding peer to service slots", zap.String("peerId", peerID.Pretty()), zap.String("service", proto))
	pm.serviceSlots[proto] = peerID
}

// SelectPeer is used to return a random peer that supports a given protocol.
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol, otherwise it will chose a peer from the service slot.
// If a peer cannot be found in the service slot, a peer will be selected from node peerstore
func (pm *PeerManager) SelectPeer(proto string, specificPeers []peer.ID, logger *zap.Logger) (peer.ID, error) {
	// @TODO We need to be more strategic about which peers we dial. Right now we just set one on the service.
	// Ideally depending on the query and our set  of peers we take a subset of ideal peers.
	// This will require us to check for various factors such as:
	//  - which topics they track
	//  - latency?
	peerSet := specificPeers
	if len(peerSet) == 0 {
		peerSet = pm.host.Peerstore().Peers()
	}

	filteredPeers := pm.filterPeersByProto(peerSet, protocol.ID(proto))

	if proto == string(WakuRelayIDv200) {
		if filteredPeers.Len() > 0 {
			pm.logger.Info("Got peer from peerstore", zap.String("peerId", filteredPeers[0].Pretty()))
			return filteredPeers[0], nil
		}
		return "", utils.ErrNoPeersAvailable
	}

	//Try to fetch from serviceSlot
	peerID, ok := pm.serviceSlots[proto]
	if ok {
		pm.logger.Info("Got peer from service slots", zap.String("peerId", peerID.Pretty()))
		return peerID, nil
	}

	if len(filteredPeers) >= 1 {
		peerID = filteredPeers[rand.Intn(len(filteredPeers))]
		pm.logger.Info("Got random peer from peerstore", zap.String("peerId", peerID.Pretty()))
		// TODO: proper heuristic here that compares peer scores and selects "best" one. For now a random peer for the given protocol is returned
		return peerID, nil // nolint: gosec
	}

	return "", utils.ErrNoPeersAvailable
}
