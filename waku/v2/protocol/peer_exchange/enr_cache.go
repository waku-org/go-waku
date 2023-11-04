package peer_exchange

import (
	"bufio"
	"bytes"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/waku-org/go-waku/waku/v2/protocol/peer_exchange/pb"
	"go.uber.org/zap"
)

// simpleLRU internal uses container/list, which is ring buffer(double linked list)
type enrCache struct {
	// using lru, saves us from periodically cleaning the cache to mauintain a certain size
	data *shardLRU
	mu   sync.RWMutex
	log  *zap.Logger
}

// err on negative size
func newEnrCache(size int, log *zap.Logger) *enrCache {
	inner := newShardLRU(int(size))
	return &enrCache{
		data: inner,
	}
}

// updating cache
func (c *enrCache) updateCache(node *enode.Node) {
	c.mu.Lock()
	defer c.mu.Unlock()
	currNode := c.data.Get(node.ID())
	if currNode == nil || node.Seq() > currNode.Seq() {
		c.data.Add(node)
		c.log.Debug("discovered px peer via discv5", zap.Stringer("enr", node))
	}
}

// get `numPeers` records of enr
func (c *enrCache) getENRs(neededPeers int, clusterIndex *ShardInfo) ([]*pb.PeerInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	//
	nodes := c.data.getRandomNodes(clusterIndex, neededPeers)
	result := []*pb.PeerInfo{}
	for _, node := range nodes {
		//
		var b bytes.Buffer
		writer := bufio.NewWriter(&b)
		err := node.Record().EncodeRLP(writer)
		if err != nil {
			return nil, err
		}
		writer.Flush()
		result = append(result, &pb.PeerInfo{
			Enr: b.Bytes(),
		})
	}
	return result, nil
}
