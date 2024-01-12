package peermanager

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/waku-org/go-waku/logging"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	waku_proto "github.com/waku-org/go-waku/waku/v2/protocol"
	"go.uber.org/zap"
)

type peersMap map[peer.ID]struct{}

// SelectPeerByContentTopic is used to return a random peer that supports a given protocol for given contentTopic.
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol and contentTopic, otherwise it will chose a peer from the service slot.
// If a peer cannot be found in the service slot, a peer will be selected from node peerstore
func (pm *PeerManager) SelectPeerByContentTopics(proto protocol.ID, contentTopics []string, specificPeers ...peer.ID) (peer.ID, error) {
	pubsubTopics := []string{}
	for _, cTopic := range contentTopics {
		pubsubTopic, err := waku_proto.GetPubSubTopicFromContentTopic(cTopic)
		if err != nil {
			pm.logger.Debug("selectPeer: failed to get contentTopic from pubsubTopic", zap.String("contentTopic", cTopic))
			return "", err
		}
		pubsubTopics = append(pubsubTopics, pubsubTopic)
	}
	peers, err := pm.SelectPeers(PeerSelectionCriteria{PubsubTopics: pubsubTopics, Proto: proto, SpecificPeers: specificPeers})
	if err != nil {
		return "", err
	}
	return peers[0], nil
}

// SelectRandomPeer is used to return a random peer that supports a given protocol.
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol, otherwise it will chose a peer from the service slot.
// If a peer cannot be found in the service slot, a peer will be selected from node peerstore
// if pubSubTopic is specified, peer is selected from list that support the pubSubTopic
func (pm *PeerManager) SelectRandom(criteria PeerSelectionCriteria) (peer.IDSlice, error) {
	// @TODO We need to be more strategic about which peers we dial. Right now we just set one on the service.
	// Ideally depending on the query and our set  of peers we take a subset of ideal peers.
	// This will require us to check for various factors such as:
	//  - which topics they track
	//  - latency?

	peerIDs, err := pm.selectServicePeer(criteria)
	if err == nil && peerIDs.Len() == criteria.MaxPeers {
		return peerIDs, nil
	} else if !errors.Is(err, ErrNoPeersAvailable) {
		pm.logger.Debug("could not retrieve random peer from slot", zap.String("protocol", string(criteria.Proto)),
			zap.Strings("pubsubTopics", criteria.PubsubTopics), zap.Error(err))
		return nil, err
	}

	// if not found in serviceSlots or proto == WakuRelayIDv200
	filteredPeers, err := pm.FilterPeersByProto(criteria.SpecificPeers, criteria.Proto)
	if err != nil {
		return nil, err
	}
	if len(criteria.PubsubTopics) > 0 {
		filteredPeers = pm.host.Peerstore().(wps.WakuPeerstore).PeersByPubSubTopics(criteria.PubsubTopics, filteredPeers...)
	}
	randomPeers, err := selectRandomPeers(filteredPeers, criteria.MaxPeers-peerIDs.Len())
	if err != nil && peerIDs.Len() == 0 {
		return nil, err
	}

	peerIDs = append(peerIDs, randomPeers...)

	return peerIDs, nil
}

// selects count random peers from list of peers
func selectRandomPeers(peers peer.IDSlice, count int) (peer.IDSlice, error) {
	filteredPeerMap := peerSliceToMap(peers)
	i := 0
	var selectedPeers peer.IDSlice
	for pID := range filteredPeerMap {
		selectedPeers = append(selectedPeers, pID)
		i++
		if i == count {
			break
		}
	}
	if selectedPeers.Len() == 0 {
		return nil, ErrNoPeersAvailable
	}
	return selectedPeers, nil
}

func peerSliceToMap(peers peer.IDSlice) peersMap {
	peerSet := make(peersMap, peers.Len())
	for _, peer := range peers {
		peerSet[peer] = struct{}{}
	}
	return peerSet
}

func (pm *PeerManager) selectServicePeer(criteria PeerSelectionCriteria) (peer.IDSlice, error) {
	var peers peer.IDSlice
	var err error
	for retryCnt := 0; retryCnt < 1; retryCnt++ {
		//Try to fetch from serviceSlot
		if slot := pm.serviceSlots.getPeers(criteria.Proto); slot != nil {
			if len(criteria.PubsubTopics) == 0 || (len(criteria.PubsubTopics) == 1 && criteria.PubsubTopics[0] == "") {
				return slot.getRandom(criteria.MaxPeers)
			} else { //PubsubTopic based selection
				keys := make([]peer.ID, 0, len(slot.m))
				for i := range slot.m {
					keys = append(keys, i)
				}
				selectedPeers := pm.host.Peerstore().(wps.WakuPeerstore).PeersByPubSubTopics(criteria.PubsubTopics, keys...)
				tmpPeers, err := selectRandomPeers(selectedPeers, criteria.MaxPeers)
				peers = append(peers, tmpPeers...)
				if err == nil && peers.Len() == criteria.MaxPeers {
					return peers, nil
				} else {
					pm.logger.Debug("discovering peers by pubsubTopic", zap.Strings("pubsubTopics", criteria.PubsubTopics))
					//Trigger on-demand discovery for this topic and connect to peer immediately.
					//For now discover atleast 1 peer for the criteria
					pm.discoverPeersByPubsubTopics(criteria.PubsubTopics, criteria.Proto, criteria.Ctx, 1)
					//Try to fetch peers again.
					continue
				}
			}
		}
	}
	if peers.Len() == 0 {
		pm.logger.Debug("could not retrieve random peer from slot", zap.Error(err))
	}
	return peers, ErrNoPeersAvailable
}

// PeerSelectionCriteria is the selection Criteria that is used by PeerManager to select peers.
type PeerSelectionCriteria struct {
	SelectionType PeerSelection
	Proto         protocol.ID
	PubsubTopics  []string
	SpecificPeers peer.IDSlice
	MaxPeers      int
	Ctx           context.Context
}

// SelectPeers selects a peer based on selectionType specified.
// Context is required only in case of selectionType set to LowestRTT
func (pm *PeerManager) SelectPeers(criteria PeerSelectionCriteria) (peer.IDSlice, error) {

	switch criteria.SelectionType {
	case Automatic:
		return pm.SelectRandom(criteria)
	case LowestRTT:
		peerID, err := pm.SelectPeerWithLowestRTT(criteria)
		if err != nil {
			return nil, err
		}
		//TODO: Update this once peer Ping cache PR is merged into this code.
		return []peer.ID{peerID}, nil
	default:
		return nil, errors.New("unknown peer selection type specified")
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
		pm.logger.Warn("context is not passed for peerSelectionwithRTT, using background context")
		criteria.Ctx = context.Background()
	}

	if len(criteria.PubsubTopics) == 0 || (len(criteria.PubsubTopics) == 1 && criteria.PubsubTopics[0] == "") {
		peers = pm.host.Peerstore().(wps.WakuPeerstore).PeersByPubSubTopics(criteria.PubsubTopics, criteria.SpecificPeers...)
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
