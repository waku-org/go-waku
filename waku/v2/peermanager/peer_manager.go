package peermanager

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/discv5"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	waku_proto "github.com/waku-org/go-waku/waku/v2/protocol"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/service"

	"go.uber.org/zap"
)

// NodeTopicDetails stores pubSubTopic related data like topicHandle for the node.
type NodeTopicDetails struct {
	topic *pubsub.Topic
}

// WakuProtoInfo holds protocol specific info
// To be used at a later stage to set various config such as criteria for peer management specific to each Waku protocols
// This should make peer-manager agnostic to protocol
type WakuProtoInfo struct {
	waku2ENRBitField uint8
}

// PeerManager applies various controls and manage connections towards peers.
type PeerManager struct {
	peerConnector          *PeerConnectionStrategy
	maxPeers               int
	maxRelayPeers          int
	logger                 *zap.Logger
	InRelayPeersTarget     int
	OutRelayPeersTarget    int
	host                   host.Host
	serviceSlots           *ServiceSlots
	ctx                    context.Context
	sub                    event.Subscription
	topicMutex             sync.RWMutex
	subRelayTopics         map[string]*NodeTopicDetails
	discoveryService       *discv5.DiscoveryV5
	wakuprotoToENRFieldMap map[protocol.ID]WakuProtoInfo
}

// PeerSelection provides various options based on which Peer is selected from a list of peers.
type PeerSelection int

const (
	Automatic PeerSelection = iota
	LowestRTT
)

// ErrNoPeersAvailable is emitted when no suitable peers are found for
// some protocol
var ErrNoPeersAvailable = errors.New("no suitable peers found")

const peerConnectivityLoopSecs = 15
const maxConnsToPeerRatio = 5

// 80% relay peers 20% service peers
func relayAndServicePeers(maxConnections int) (int, int) {
	return maxConnections - maxConnections/5, maxConnections / 5
}

// 66% inRelayPeers 33% outRelayPeers
func inAndOutRelayPeers(relayPeers int) (int, int) {
	outRelayPeers := relayPeers / 3
	//
	const minOutRelayConns = 10
	if outRelayPeers < minOutRelayConns {
		outRelayPeers = minOutRelayConns
	}
	return relayPeers - outRelayPeers, outRelayPeers
}

// NewPeerManager creates a new peerManager instance.
func NewPeerManager(maxConnections int, maxPeers int, logger *zap.Logger) *PeerManager {

	maxRelayPeers, _ := relayAndServicePeers(maxConnections)
	inRelayPeersTarget, outRelayPeersTarget := inAndOutRelayPeers(maxRelayPeers)

	if maxPeers == 0 || maxConnections > maxPeers {
		maxPeers = maxConnsToPeerRatio * maxConnections
	}

	pm := &PeerManager{
		logger:                 logger.Named("peer-manager"),
		maxRelayPeers:          maxRelayPeers,
		InRelayPeersTarget:     inRelayPeersTarget,
		OutRelayPeersTarget:    outRelayPeersTarget,
		serviceSlots:           NewServiceSlot(),
		subRelayTopics:         make(map[string]*NodeTopicDetails),
		maxPeers:               maxPeers,
		wakuprotoToENRFieldMap: map[protocol.ID]WakuProtoInfo{},
	}
	logger.Info("PeerManager init values", zap.Int("maxConnections", maxConnections),
		zap.Int("maxRelayPeers", maxRelayPeers),
		zap.Int("outRelayPeersTarget", outRelayPeersTarget),
		zap.Int("inRelayPeersTarget", pm.InRelayPeersTarget),
		zap.Int("maxPeers", maxPeers))

	return pm
}

func (pm *PeerManager) SetDiscv5(discv5 *discv5.DiscoveryV5) {
	pm.discoveryService = discv5
}

// SetHost sets the host to be used in order to access the peerStore.
func (pm *PeerManager) SetHost(host host.Host) {
	pm.host = host
}

// SetPeerConnector sets the peer connector to be used for establishing relay connections.
func (pm *PeerManager) SetPeerConnector(pc *PeerConnectionStrategy) {
	pm.peerConnector = pc
}

// Start starts the processing to be done by peer manager.
func (pm *PeerManager) Start(ctx context.Context) {

	var enrField uint8
	enrField |= (1 << 0)
	pm.RegisterWakuProtocol(relay.WakuRelayID_v200, enrField)

	pm.ctx = ctx
	if pm.sub != nil {
		go pm.peerEventLoop(ctx)
	}
	go pm.connectivityLoop(ctx)
}

// This is a connectivity loop, which currently checks and prunes inbound connections.
func (pm *PeerManager) connectivityLoop(ctx context.Context) {
	pm.connectToRelayPeers()
	t := time.NewTicker(peerConnectivityLoopSecs * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pm.connectToRelayPeers()
		}
	}
}

// GroupPeersByDirection returns all the connected peers in peer store grouped by Inbound or outBound direction
func (pm *PeerManager) GroupPeersByDirection(specificPeers ...peer.ID) (inPeers peer.IDSlice, outPeers peer.IDSlice, err error) {
	if len(specificPeers) == 0 {
		specificPeers = pm.host.Network().Peers()
	}

	for _, p := range specificPeers {
		direction, err := pm.host.Peerstore().(wps.WakuPeerstore).Direction(p)
		if err == nil {
			if direction == network.DirInbound {
				inPeers = append(inPeers, p)
			} else if direction == network.DirOutbound {
				outPeers = append(outPeers, p)
			}
		} else {
			pm.logger.Error("Failed to retrieve peer direction",
				logging.HostID("peerID", p), zap.Error(err))
		}
	}
	return inPeers, outPeers, nil
}

// getRelayPeers - Returns list of in and out peers supporting WakuRelayProtocol within specifiedPeers.
// If specifiedPeers is empty, it checks within all peers in peerStore.
func (pm *PeerManager) getRelayPeers(specificPeers ...peer.ID) (inRelayPeers peer.IDSlice, outRelayPeers peer.IDSlice) {
	//Group peers by their connected direction inbound or outbound.
	inPeers, outPeers, err := pm.GroupPeersByDirection(specificPeers...)
	if err != nil {
		return
	}
	pm.logger.Debug("Number of peers connected", zap.Int("inPeers", inPeers.Len()),
		zap.Int("outPeers", outPeers.Len()))

	//Need to filter peers to check if they support relay
	if inPeers.Len() != 0 {
		inRelayPeers, _ = pm.FilterPeersByProto(inPeers, relay.WakuRelayID_v200)
	}
	if outPeers.Len() != 0 {
		outRelayPeers, _ = pm.FilterPeersByProto(outPeers, relay.WakuRelayID_v200)
	}
	return
}

func (pm *PeerManager) DiscoverAndConnectToPeers(ctx context.Context, cluster uint16,
	shard uint16, serviceProtocol protocol.ID) error {
	peers, err := pm.discoverOnDemand(cluster, shard, serviceProtocol)
	if err != nil {
		return err
	}
	pm.logger.Debug("discovered peers on demand ", zap.Int("noOfPeers", len(peers)))
	connectNow := false
	//Add discovered peers to peerStore and connect to them
	for idx, p := range peers {
		if serviceProtocol != relay.WakuRelayID_v200 && idx <= 1 {
			//how many connections to initiate? Maybe this could be a config exposed to client API.
			//For now just going ahead with initiating connections with 2 nodes in case of non-relay service
			//In case of relay let it go through connectivityLoop
			connectNow = true
		}
		pm.AddDiscoveredPeer(p, connectNow)
	}
	return nil
}

// RegisterWakuProtocol to be used by Waku protocols that could be used for peer discovery
// Which means protoocl should be as defined in waku2 ENR key in https://rfc.vac.dev/spec/31/.
func (pm *PeerManager) RegisterWakuProtocol(proto protocol.ID, bitField uint8) {
	pm.wakuprotoToENRFieldMap[proto] = WakuProtoInfo{waku2ENRBitField: bitField}
}

//type Predicate func(enode.Iterator) enode.Iterator

// OnDemandPeerDiscovery initiates an on demand peer discovery and
// filters peers based on cluster,shard and any wakuservice protocols specified
func (pm *PeerManager) discoverOnDemand(cluster uint16,
	shard uint16, wakuProtocol protocol.ID) ([]service.PeerData, error) {
	var peers []service.PeerData

	wakuProtoInfo, ok := pm.wakuprotoToENRFieldMap[wakuProtocol]
	if !ok {
		pm.logger.Error("cannot do on demand discovery for non-waku protocol", zap.String("protocol", string(wakuProtocol)))
		return nil, errors.New("cannot do on demand discovery for non-waku protocol")
	}

	iterator, err := pm.discoveryService.PeerIterator(
		discv5.FilterShard(cluster, shard),
		discv5.FilterCapabilities(wakuProtoInfo.waku2ENRBitField))
	if err != nil {
		pm.logger.Error("failed to find peers for shard and services", zap.Uint16("cluster", cluster),
			zap.Uint16("shard", shard), zap.String("service", string(wakuProtocol)), zap.Error(err))
		return peers, err
	}

	//Iterate and fill peers.
	defer iterator.Close()

	for iterator.Next() {
		pInfo, err := wenr.EnodeToPeerInfo(iterator.Node())
		if err != nil {
			continue
		}
		pData := service.PeerData{
			Origin:   wps.Discv5,
			ENR:      iterator.Node(),
			AddrInfo: *pInfo,
		}
		peers = append(peers, pData)

	}
	return peers, nil
}

func (pm *PeerManager) discoverPeersByPubsubTopic(pubsubTopic string, proto protocol.ID) {
	shardInfo, err := waku_proto.TopicsToRelayShards(pubsubTopic)
	if err != nil {
		pm.logger.Error("failed to convert pubsub topic to shard", zap.String("topic", pubsubTopic), zap.Error(err))
		return
	}
	pm.DiscoverAndConnectToPeers(pm.ctx, shardInfo[0].Cluster, shardInfo[0].Indices[0], proto)
}

// ensureMinRelayConnsPerTopic makes sure there are min of D conns per pubsubTopic.
// If not it will look into peerStore to initiate more connections.
// If peerStore doesn't have enough peers, will wait for discv5 to find more and try in next cycle
func (pm *PeerManager) ensureMinRelayConnsPerTopic() {
	pm.topicMutex.RLock()
	defer pm.topicMutex.RUnlock()
	for topicStr, topicInst := range pm.subRelayTopics {
		curPeers := topicInst.topic.ListPeers()
		curPeerLen := len(curPeers)
		if curPeerLen < waku_proto.GossipSubOptimalFullMeshSize {
			pm.logger.Info("Subscribed topic is unhealthy, initiating more connections to maintain health",
				zap.String("pubSubTopic", topicStr), zap.Int("connectedPeerCount", curPeerLen),
				zap.Int("optimumPeers", waku_proto.GossipSubOptimalFullMeshSize))
			//Find not connected peers.
			notConnectedPeers := pm.getNotConnectedPers(topicStr)
			if notConnectedPeers.Len() == 0 {
				pm.discoverPeersByPubsubTopic(topicStr, relay.WakuRelayID_v200)
				continue
			}
			//Connect to eligible peers.
			numPeersToConnect := waku_proto.GossipSubOptimalFullMeshSize - curPeerLen

			if numPeersToConnect > notConnectedPeers.Len() {
				numPeersToConnect = notConnectedPeers.Len()
			}
			pm.connectToPeers(notConnectedPeers[0:numPeersToConnect])
		}
	}
}

// connectToRelayPeers ensures minimum D connections are there for each pubSubTopic.
// If not, initiates connections to additional peers.
// It also checks for incoming relay connections and prunes once they cross inRelayTarget
func (pm *PeerManager) connectToRelayPeers() {
	//Check for out peer connections and connect to more peers.
	pm.ensureMinRelayConnsPerTopic()

	inRelayPeers, outRelayPeers := pm.getRelayPeers()
	pm.logger.Info("number of relay peers connected",
		zap.Int("in", inRelayPeers.Len()),
		zap.Int("out", outRelayPeers.Len()))
	if inRelayPeers.Len() > 0 &&
		inRelayPeers.Len() > pm.InRelayPeersTarget {
		pm.pruneInRelayConns(inRelayPeers)
	}
}

// addrInfoToPeerData returns addressinfo for a peer
// If addresses are expired, it removes the peer from host peerStore and returns nil.
func addrInfoToPeerData(origin wps.Origin, peerID peer.ID, host host.Host) *service.PeerData {
	addrs := host.Peerstore().Addrs(peerID)
	if len(addrs) == 0 {
		//Addresses expired, remove peer from peerStore
		host.Peerstore().RemovePeer(peerID)
		return nil
	}
	return &service.PeerData{
		Origin: origin,
		AddrInfo: peer.AddrInfo{
			ID:    peerID,
			Addrs: addrs,
		},
	}
}

// connectToPeers connects to peers provided in the list if the addresses have not expired.
func (pm *PeerManager) connectToPeers(peers peer.IDSlice) {
	for _, peerID := range peers {
		peerData := addrInfoToPeerData(wps.PeerManager, peerID, pm.host)
		if peerData == nil {
			continue
		}
		pm.peerConnector.PushToChan(*peerData)
	}
}

// getNotConnectedPers returns peers for a pubSubTopic that are not connected.
func (pm *PeerManager) getNotConnectedPers(pubsubTopic string) (notConnectedPeers peer.IDSlice) {
	var peerList peer.IDSlice
	if pubsubTopic == "" {
		peerList = pm.host.Peerstore().Peers()
	} else {
		peerList = pm.host.Peerstore().(*wps.WakuPeerstoreImpl).PeersByPubSubTopic(pubsubTopic)
	}
	for _, peerID := range peerList {
		if pm.host.Network().Connectedness(peerID) != network.Connected {
			notConnectedPeers = append(notConnectedPeers, peerID)
		}
	}
	return
}

// pruneInRelayConns prune any incoming relay connections crossing derived inrelayPeerTarget
func (pm *PeerManager) pruneInRelayConns(inRelayPeers peer.IDSlice) {

	//Start disconnecting peers, based on what?
	//For now no preference is used
	//TODO: Need to have more intelligent way of doing this, maybe peer scores.
	//TODO: Keep optimalPeersRequired for a pubSubTopic in mind while pruning connections to peers.
	pm.logger.Info("peer connections exceed target relay peers, hence pruning",
		zap.Int("cnt", inRelayPeers.Len()), zap.Int("target", pm.InRelayPeersTarget))
	for pruningStartIndex := pm.InRelayPeersTarget; pruningStartIndex < inRelayPeers.Len(); pruningStartIndex++ {
		p := inRelayPeers[pruningStartIndex]
		err := pm.host.Network().ClosePeer(p)
		if err != nil {
			pm.logger.Warn("Failed to disconnect connection towards peer",
				logging.HostID("peerID", p))
		}
		pm.logger.Debug("Successfully disconnected connection towards peer",
			logging.HostID("peerID", p))
	}
}

// AddDiscoveredPeer to add dynamically discovered peers.
// Note that these peers will not be set in service-slots.
// TODO: It maybe good to set in service-slots based on services supported in the ENR
func (pm *PeerManager) AddDiscoveredPeer(p service.PeerData, connectNow bool) {
	//Doing this check again inside addPeer, in order to avoid additional complexity of rollingBack other changes.
	if pm.maxPeers <= pm.host.Peerstore().Peers().Len() {
		return
	}
	//Check if the peer is already present, if so skip adding
	_, err := pm.host.Peerstore().(wps.WakuPeerstore).Origin(p.AddrInfo.ID)
	if err == nil {
		pm.logger.Debug("Found discovered peer already in peerStore", logging.HostID("peer", p.AddrInfo.ID))
		return
	}
	// Try to fetch shard info from ENR to arrive at pubSub topics.
	if len(p.PubSubTopics) == 0 && p.ENR != nil {
		shards, err := wenr.RelaySharding(p.ENR.Record())
		if err != nil {
			pm.logger.Error("Could not derive relayShards from ENR", zap.Error(err),
				logging.HostID("peer", p.AddrInfo.ID), zap.String("enr", p.ENR.String()))
		} else {
			if shards != nil {
				p.PubSubTopics = make([]string, 0)
				topics := shards.Topics()
				for _, topic := range topics {
					topicStr := topic.String()
					p.PubSubTopics = append(p.PubSubTopics, topicStr)
				}
			} else {
				pm.logger.Debug("ENR doesn't have relay shards", logging.HostID("peer", p.AddrInfo.ID))
			}
		}
	}

	_ = pm.addPeer(p.AddrInfo.ID, p.AddrInfo.Addrs, p.Origin, p.PubSubTopics)

	if p.ENR != nil {
		err := pm.host.Peerstore().(wps.WakuPeerstore).SetENR(p.AddrInfo.ID, p.ENR)
		if err != nil {
			pm.logger.Error("could not store enr", zap.Error(err),
				logging.HostID("peer", p.AddrInfo.ID), zap.String("enr", p.ENR.String()))
		}
	}
	if connectNow {
		go pm.peerConnector.PushToChan(p)
	}
}

// addPeer adds peer to only the peerStore.
// It also sets additional metadata such as origin, ENR and supported protocols
func (pm *PeerManager) addPeer(ID peer.ID, addrs []ma.Multiaddr, origin wps.Origin, pubSubTopics []string, protocols ...protocol.ID) error {
	if pm.maxPeers <= pm.host.Peerstore().Peers().Len() {
		return errors.New("peer store capacity reached")
	}
	pm.logger.Info("adding peer to peerstore", logging.HostID("peer", ID))
	if origin == wps.Static {
		pm.host.Peerstore().AddAddrs(ID, addrs, peerstore.PermanentAddrTTL)
	} else {
		//Need to re-evaluate the address expiry
		// For now expiring them with default addressTTL which is an hour.
		pm.host.Peerstore().AddAddrs(ID, addrs, peerstore.AddressTTL)
	}
	err := pm.host.Peerstore().(wps.WakuPeerstore).SetOrigin(ID, origin)
	if err != nil {
		pm.logger.Error("could not set origin", zap.Error(err), logging.HostID("peer", ID))
		return err
	}

	if len(protocols) > 0 {
		err = pm.host.Peerstore().AddProtocols(ID, protocols...)
		if err != nil {
			return err
		}
	}
	if len(pubSubTopics) == 0 {
		// Probably the peer is discovered via DNSDiscovery (for which we don't have pubSubTopic info)
		//If pubSubTopic and enr is empty or no shard info in ENR,then set to defaultPubSubTopic
		pubSubTopics = []string{relay.DefaultWakuTopic}
	}
	err = pm.host.Peerstore().(wps.WakuPeerstore).SetPubSubTopics(ID, pubSubTopics)
	if err != nil {
		pm.logger.Error("could not store pubSubTopic", zap.Error(err),
			logging.HostID("peer", ID), zap.Strings("topics", pubSubTopics))
	}
	return nil
}

// AddPeer adds peer to the peerStore and also to service slots
func (pm *PeerManager) AddPeer(address ma.Multiaddr, origin wps.Origin, pubSubTopics []string, protocols ...protocol.ID) (peer.ID, error) {
	//Assuming all addresses have peerId
	info, err := peer.AddrInfoFromP2pAddr(address)
	if err != nil {
		return "", err
	}

	//Add Service peers to serviceSlots.
	for _, proto := range protocols {
		pm.addPeerToServiceSlot(proto, info.ID)
	}

	//Add to the peer-store
	err = pm.addPeer(info.ID, info.Addrs, origin, pubSubTopics, protocols...)
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
	pm.serviceSlots.removePeer(peerID)
}

// addPeerToServiceSlot adds a peerID to serviceSlot.
// Adding to peerStore is expected to be already done by caller.
// If relay proto is passed, it is not added to serviceSlot.
func (pm *PeerManager) addPeerToServiceSlot(proto protocol.ID, peerID peer.ID) {
	if proto == relay.WakuRelayID_v200 {
		pm.logger.Warn("Cannot add Relay peer to service peer slots")
		return
	}

	//For now adding the peer to serviceSlot which means the latest added peer would be given priority.
	//TODO: Ideally we should sort the peers per service and return best peer based on peer score or RTT etc.
	pm.logger.Info("Adding peer to service slots", logging.HostID("peer", peerID),
		zap.String("service", string(proto)))
	// getPeers returns nil for WakuRelayIDv200 protocol, but we don't run this ServiceSlot code for WakuRelayIDv200 protocol
	pm.serviceSlots.getPeers(proto).add(peerID)
}

// SelectPeerByContentTopic is used to return a random peer that supports a given protocol for given contentTopic.
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol and contentTopic, otherwise it will chose a peer from the service slot.
// If a peer cannot be found in the service slot, a peer will be selected from node peerstore
func (pm *PeerManager) SelectPeerByContentTopic(proto protocol.ID, contentTopic string, specificPeers ...peer.ID) (peer.ID, error) {
	pubsubTopic, err := waku_proto.GetPubSubTopicFromContentTopic(contentTopic)
	if err != nil {
		return "", err
	}
	return pm.SelectPeer(PeerSelectionCriteria{PubsubTopic: pubsubTopic, Proto: proto, SpecificPeers: specificPeers})
}

// SelectRandomPeer is used to return a random peer that supports a given protocol.
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol, otherwise it will chose a peer from the service slot.
// If a peer cannot be found in the service slot, a peer will be selected from node peerstore
// if pubSubTopic is specified, peer is selected from list that support the pubSubTopic
func (pm *PeerManager) SelectRandomPeer(criteria PeerSelectionCriteria) (peer.ID, error) {
	// @TODO We need to be more strategic about which peers we dial. Right now we just set one on the service.
	// Ideally depending on the query and our set  of peers we take a subset of ideal peers.
	// This will require us to check for various factors such as:
	//  - which topics they track
	//  - latency?

	if peerID := pm.selectServicePeer(criteria.Proto, criteria.PubsubTopic, criteria.SpecificPeers...); peerID != nil {
		return *peerID, nil
	}

	// if not found in serviceSlots or proto == WakuRelayIDv200
	filteredPeers, err := pm.FilterPeersByProto(criteria.SpecificPeers, criteria.Proto)
	if err != nil {
		return "", err
	}
	if criteria.PubsubTopic != "" {
		filteredPeers = pm.host.Peerstore().(wps.WakuPeerstore).PeersByPubSubTopic(criteria.PubsubTopic, filteredPeers...)
	}
	return selectRandomPeer(filteredPeers, pm.logger)
}

func (pm *PeerManager) selectServicePeer(proto protocol.ID, pubSubTopic string, specificPeers ...peer.ID) (peerIDPtr *peer.ID) {
	peerIDPtr = nil

	//Try to fetch from serviceSlot
	if slot := pm.serviceSlots.getPeers(proto); slot != nil {
		if pubSubTopic == "" {
			if peerID, err := slot.getRandom(); err == nil {
				peerIDPtr = &peerID
			} else {
				pm.logger.Debug("could not retrieve random peer from slot", zap.Error(err))
			}
		} else { //PubsubTopic based selection
			keys := make([]peer.ID, 0, len(slot.m))
			for i := range slot.m {
				keys = append(keys, i)
			}
			selectedPeers := pm.host.Peerstore().(wps.WakuPeerstore).PeersByPubSubTopic(pubSubTopic, keys...)
			peerID, err := selectRandomPeer(selectedPeers, pm.logger)
			if err == nil {
				peerIDPtr = &peerID
			} else {
				//TODO:Trigger on-demand discovery for this topic and connect to peer immediately and set peerID.
				//TODO: Use context to limit time for connectivity
				//pm.discoverPeersByPubsubTopic(pubSubTopic, proto)
				pm.logger.Debug("could not select random peer", zap.Error(err))
			}
		}
	}
	return
}

// PeerSelectionCriteria is the selection Criteria that is used by PeerManager to select peers.
type PeerSelectionCriteria struct {
	SelectionType PeerSelection
	Proto         protocol.ID
	PubsubTopic   string
	SpecificPeers peer.IDSlice
	Ctx           context.Context
}

// SelectPeer selects a peer based on selectionType specified.
// Context is required only in case of selectionType set to LowestRTT
func (pm *PeerManager) SelectPeer(criteria PeerSelectionCriteria) (peer.ID, error) {

	switch criteria.SelectionType {
	case Automatic:
		return pm.SelectRandomPeer(criteria)
	case LowestRTT:
		if criteria.Ctx == nil {
			criteria.Ctx = context.Background()
			pm.logger.Warn("context is not passed for peerSelectionwithRTT, using background context")
		}
		return pm.SelectPeerWithLowestRTT(criteria)
	default:
		return "", errors.New("unknown peer selection type specified")
	}
}

type pingResult struct {
	p   peer.ID
	rtt time.Duration
}

// SelectPeerWithLowestRTT will select a peer that supports a specific protocol with the lowest reply time
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol, otherwise it will chose a peer from the node peerstore
// TO OPTIMIZE: As of now the peer with lowest RTT is identified when select is called, this should be optimized
// to maintain the RTT as part of peer-scoring and just select based on that.
func (pm *PeerManager) SelectPeerWithLowestRTT(criteria PeerSelectionCriteria) (peer.ID, error) {
	var peers peer.IDSlice
	var err error
	if criteria.Ctx == nil {
		criteria.Ctx = context.Background()
	}

	if criteria.PubsubTopic != "" {
		peers = pm.host.Peerstore().(wps.WakuPeerstore).PeersByPubSubTopic(criteria.PubsubTopic, criteria.SpecificPeers...)
	}

	peers, err = pm.FilterPeersByProto(peers, criteria.Proto)
	if err != nil {
		return "", err
	}
	wg := sync.WaitGroup{}
	waitCh := make(chan struct{})
	pingCh := make(chan pingResult, 1000)

	wg.Add(len(peers))

	go func() {
		for _, p := range peers {
			go func(p peer.ID) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(criteria.Ctx, 3*time.Second)
				defer cancel()
				result := <-ping.Ping(ctx, pm.host, p)
				if result.Error == nil {
					pingCh <- pingResult{
						p:   p,
						rtt: result.RTT,
					}
				} else {
					pm.logger.Debug("could not ping", logging.HostID("peer", p), zap.Error(result.Error))
				}
			}(p)
		}
		wg.Wait()
		close(waitCh)
		close(pingCh)
	}()

	select {
	case <-waitCh:
		var min *pingResult
		for p := range pingCh {
			if min == nil {
				min = &p
			} else {
				if p.rtt < min.rtt {
					min = &p
				}
			}
		}
		if min == nil {
			return "", ErrNoPeersAvailable
		}

		return min.p, nil
	case <-criteria.Ctx.Done():
		return "", ErrNoPeersAvailable
	}
}

// selectRandomPeer selects randomly a peer from the list of peers passed.
func selectRandomPeer(peers peer.IDSlice, log *zap.Logger) (peer.ID, error) {
	if len(peers) >= 1 {
		peerID := peers[rand.Intn(len(peers))]
		// TODO: proper heuristic here that compares peer scores and selects "best" one. For now a random peer for the given protocol is returned
		return peerID, nil // nolint: gosec
	}

	return "", ErrNoPeersAvailable
}

// FilterPeersByProto filters list of peers that support specified protocols.
// If specificPeers is nil, all peers in the host's peerStore are considered for filtering.
func (pm *PeerManager) FilterPeersByProto(specificPeers peer.IDSlice, proto ...protocol.ID) (peer.IDSlice, error) {
	peerSet := specificPeers
	if len(peerSet) == 0 {
		peerSet = pm.host.Peerstore().Peers()
	}

	var peers peer.IDSlice
	for _, peer := range peerSet {
		protocols, err := pm.host.Peerstore().SupportsProtocols(peer, proto...)
		if err != nil {
			return nil, err
		}

		if len(protocols) > 0 {
			peers = append(peers, peer)
		}
	}
	return peers, nil
}
