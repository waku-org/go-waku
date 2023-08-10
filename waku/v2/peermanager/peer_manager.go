package peermanager

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/logging"
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
	serviceSlots        map[protocol.ID]peer.ID
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
		serviceSlots:        make(map[protocol.ID]peer.ID),
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

func (pm *PeerManager) pruneInRelayConns() {

	var inRelayPeers peer.IDSlice
	//Group peers by their connected direction inbound or outbound.
	inPeers, outPeers, err := pm.host.Peerstore().(wps.WakuPeerstore).GroupPeersByDirection()
	if err != nil {
		return
	}
	pm.logger.Info("Number of peers connected", zap.Int("inPeers", inPeers.Len()), zap.Int("outPeers", outPeers.Len()))

	//Need to filter peers to check if they support relay
	inRelayPeers, _ = utils.FilterPeersByProto(pm.host, inPeers, WakuRelayIDv200)
	outRelayPeers, _ := utils.FilterPeersByProto(pm.host, outPeers, WakuRelayIDv200)
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

// AddPeer adds peer to the peerStore and also to service slots
func (pm *PeerManager) AddPeer(address ma.Multiaddr, origin wps.Origin, protocols ...protocol.ID) (peer.ID, error) {
	info, err := peer.AddrInfoFromP2pAddr(address)
	if err != nil {
		return "", err
	}

	//Add Service peers to serviceSlots.
	for _, service := range protocols {
		pm.AddPeerToServiceSlot(service, info.ID)
	}
	//Set origin, ENR and Direction and any other required items in Waku Peer Store.
	pm.logger.Info("adding peer to peerstore", logging.HostID("peer", info.ID))
	err = pm.host.Peerstore().(wps.WakuPeerstore).SetOrigin(info.ID, origin)
	if err != nil {
		return "", err
	}

	pm.host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.AddressTTL)
	err = pm.host.Peerstore().AddProtocols(info.ID, protocols...)
	if err != nil {
		return "", err
	}
	return info.ID, nil
}

// RemovePeer deletes peer from the peerStore after disconnecting it.
// It also removes the peer from serviceSlot.
func (pm *PeerManager) RemovePeer(peerID peer.ID) {
	pm.host.Peerstore().RemovePeer(peerID)
	//Search if this peer is in serviceSlot and if so, remove it from there
	// TODO:Add another peer which is statically configured to the serviceSlot.
	for proto, peer := range pm.serviceSlots {
		if peer == peerID {
			pm.serviceSlots[proto] = ""
		}
	}
}

// AddServicePeer adds a peerID to serviceSlot.
// Adding to peerStore is expected to be already done by caller.
// If relay proto is passed, it is not added to serviceSlot.
func (pm *PeerManager) AddPeerToServiceSlot(proto protocol.ID, peerID peer.ID) {
	if proto == WakuRelayIDv200 {
		pm.logger.Warn("Cannot add Relay peer to service peer slots")
		return
	}
	//For now adding the peer to serviceSlot which means the latest added peer would be given priority.
	//TODO: Ideally we should maintain multiple peers per service and return best peer based on peer score or RTT etc.
	//Should we override service slot if peer is identified dynamically?
	pm.logger.Info("Adding peer to service slots", logging.HostID("peer", peerID), zap.String("service", string(proto)))
	pm.serviceSlots[proto] = peerID
}

// SelectPeer is used to return a random peer that supports a given protocol.
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol, otherwise it will chose a peer from the service slot.
// If a peer cannot be found in the service slot, a peer will be selected from node peerstore
func (pm *PeerManager) SelectPeer(proto protocol.ID, specificPeers []peer.ID, logger *zap.Logger) (peer.ID, error) {
	// @TODO We need to be more strategic about which peers we dial. Right now we just set one on the service.
	// Ideally depending on the query and our set  of peers we take a subset of ideal peers.
	// This will require us to check for various factors such as:
	//  - which topics they track
	//  - latency?

	filteredPeers, err := utils.FilterPeersByProto(pm.host, specificPeers, proto)
	if err != nil {
		return "", err
	}
	if proto == WakuRelayIDv200 {
		return utils.SelectRandomPeer(filteredPeers, pm.logger)
	}

	//Try to fetch from serviceSlot
	peerID, ok := pm.serviceSlots[proto]
	if ok || peerID != "" {
		pm.logger.Info("Got peer from service slots", logging.HostID("peer", peerID))
		return peerID, nil
	}

	return utils.SelectRandomPeer(filteredPeers, pm.logger)
}
