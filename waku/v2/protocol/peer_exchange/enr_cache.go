package peer_exchange

import (
	"bufio"
	"bytes"
	"math/rand"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/waku-org/go-waku/waku/v2/protocol/peer_exchange/pb"
	"go.uber.org/zap"
)

// simpleLRU internal uses container/list, which is ring buffer(double linked list)
type enrCache struct {
	// using lru, saves us from periodically cleaning the cache to mauint16ain a certain size
	data *shardLRU
	rng  *rand.Rand
	mu   sync.RWMutex
	log  *zap.Logger
}

// err on negative size
func newEnrCache(size int, log *zap.Logger) *enrCache {
	inner := newShardLRU(int(size))
	return &enrCache{
		data: inner,
		rng:  rand.New(rand.NewSource(rand.Int63())),
		log:  log.Named("enr-cache"),
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
func (c *enrCache) getENRs(neededPeers int, clusterIndex *ClusterShard) ([]*pb.PeerInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	//
	availablePeers := c.data.len(clusterIndex)
	if availablePeers == 0 {
		return nil, nil
	}
	if availablePeers < neededPeers {
		neededPeers = availablePeers
	}

	perm := c.rng.Perm(availablePeers)[0:neededPeers]
	nodes := c.data.getNodes(clusterIndex)
	result := []*pb.PeerInfo{}
	for _, ind := range perm {
		node := nodes[ind]
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
