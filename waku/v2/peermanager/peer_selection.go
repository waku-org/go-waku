package peermanager

import (
	"context"
	"errors"
	"math/rand"
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

	if peerID := pm.selectServicePeer(criteria.Proto, criteria.PubsubTopic, criteria.Ctx, criteria.SpecificPeers...); peerID != nil {
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

func (pm *PeerManager) selectServicePeer(proto protocol.ID, pubSubTopic string, ctx context.Context, specificPeers ...peer.ID) (peerIDPtr *peer.ID) {
	peerIDPtr = nil
	var err error
	for tries := 0; tries <= 1; tries++ {
		//Try to fetch from serviceSlot
		if slot := pm.serviceSlots.getPeers(proto); slot != nil {
			if pubSubTopic == "" {
				if peerID, err := slot.getRandom(); err == nil {
					peerIDPtr = &peerID
				}
				break
			} else { //PubsubTopic based selection
				keys := make([]peer.ID, 0, len(slot.m))
				for i := range slot.m {
					keys = append(keys, i)
				}
				selectedPeers := pm.host.Peerstore().(wps.WakuPeerstore).PeersByPubSubTopic(pubSubTopic, keys...)
				peerID, err := selectRandomPeer(selectedPeers, pm.logger)
				if err == nil {
					peerIDPtr = &peerID
					break
				} else {
					//Trigger on-demand discovery for this topic and connect to peer immediately.
					//For now discover atleast 1 peer for the criteria
					pm.discoverPeersByPubsubTopic(pubSubTopic, proto, ctx, 1)
					//Try to fetch peers again.
					continue
				}
			}
		}
	}
	if peerIDPtr == nil {
		pm.logger.Debug("could not retrieve random peer from slot", zap.Error(err))
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
