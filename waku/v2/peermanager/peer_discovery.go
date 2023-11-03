package peermanager

import (
	"context"
	"errors"

	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/waku-org/go-waku/waku/v2/discv5"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	waku_proto "github.com/waku-org/go-waku/waku/v2/protocol"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/service"
	"go.uber.org/zap"
)

// DiscoverAndConnectToPeers discovers peers using discoveryv5 and connects to the peers.
// It discovers peers till maxCount peers are found for the cluster,shard and protocol or the context passed expires.
func (pm *PeerManager) DiscoverAndConnectToPeers(ctx context.Context, cluster uint16,
	shard uint16, serviceProtocol protocol.ID, maxCount int) error {
	if pm.discoveryService == nil {
		return nil
	}
	peers, err := pm.discoverOnDemand(cluster, shard, serviceProtocol, ctx, maxCount)
	if err != nil {
		return err
	}
	pm.logger.Debug("discovered peers on demand ", zap.Int("noOfPeers", len(peers)))
	connectNow := false
	//Add discovered peers to peerStore and connect to them
	for idx, p := range peers {
		if serviceProtocol != relay.WakuRelayID_v200 && idx <= maxCount {
			//how many connections to initiate? Maybe this could be a config exposed to client API.
			//For now just going ahead with initiating connections with 2 nodes in case of non-relay service peers
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
	shard uint16, wakuProtocol protocol.ID, ctx context.Context, maxCount int) ([]service.PeerData, error) {
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
		if ctx.Err() != nil || len(peers) >= maxCount {
			break
		}
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

func (pm *PeerManager) discoverPeersByPubsubTopic(pubsubTopic string, proto protocol.ID, ctx context.Context, maxCount int) {
	shardInfo, err := waku_proto.TopicsToRelayShards(pubsubTopic)
	if err != nil {
		pm.logger.Error("failed to convert pubsub topic to shard", zap.String("topic", pubsubTopic), zap.Error(err))
		return
	}
	err = pm.DiscoverAndConnectToPeers(ctx, shardInfo[0].ClusterID, shardInfo[0].ShardIDs[0], proto, maxCount)
	if err != nil {
		pm.logger.Error("failed to discover and conenct to peers", zap.Error(err))
	}
}
