package peer_exchange

import (
	"crypto/ecdsa"
	"testing"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
)

func getEnode(t *testing.T, key *ecdsa.PrivateKey, cluster uint16, indexes ...uint16) *enode.Node {
	if key == nil {
		var err error
		key, err = gcrypto.GenerateKey()
		require.NoError(t, err)
	}
	myNode, err := wenr.NewLocalnode(key)
	require.NoError(t, err)
	if cluster != 0 {
		shard, err := protocol.NewRelayShards(cluster, indexes...)
		require.NoError(t, err)
		myNode.Set(enr.WithEntry(wenr.ShardingBitVectorEnrField, shard.BitVector()))
	}
	return myNode.Node()
}

func TestLruMoreThanSize(t *testing.T) {
	node1 := getEnode(t, nil, 1, 1)
	node2 := getEnode(t, nil, 1, 1, 2)
	node3 := getEnode(t, nil, 1, 1)

	lru := newShardLRU(2)

	lru.Add(node1)
	lru.Add(node2)
	lru.Add(node3)

	nodes := lru.getNodes(&ClusterIndex{1, 1})

	require.Equal(t, 2, len(nodes))
	require.Equal(t, node2.ID(), nodes[0].ID())
	require.Equal(t, node3.ID(), nodes[1].ID())
	// checks if removed node is deleted from lru or not
	require.Nil(t, lru.Get(node1.ID()))

	// node 2 is removed from lru for cluster 1/index 1 but it is still being maintained for cluster1,index2
	node4 := getEnode(t, nil, 1, 1, 2)
	lru.Add(node4)

	// node2 still present in lru
	require.Equal(t, node2, lru.Get(node2.ID()))

	// now node2 is removed from all shards' cache
	node5 := getEnode(t, nil, 1, 2)
	lru.Add(node5)

	// node2 still present in lru
	require.Nil(t, lru.Get(node2.ID()))
}

func TestLruNodeWithNewSeq(t *testing.T) {
	lru := newShardLRU(2)
	//
	key, err := gcrypto.GenerateKey()
	require.NoError(t, err)

	node1 := getEnode(t, key, 1, 1)
	lru.Add(node1)

	node1 = getEnode(t, key, 1, 2, 3)
	lru.Add(node1)

	//
	nodes := lru.getNodes(&ClusterIndex{1, 1})
	require.Equal(t, 0, len(nodes))
	//
	nodes = lru.getNodes(&ClusterIndex{1, 2})
	require.Equal(t, 1, len(nodes))
	//
	nodes = lru.getNodes(&ClusterIndex{1, 3})
	require.Equal(t, 1, len(nodes))
}

func TestLruNoShard(t *testing.T) {
	lru := newShardLRU(2)

	node1 := getEnode(t, nil, 0)
	node2 := getEnode(t, nil, 0)

	lru.Add(node1)
	lru.Add(node2)

	// check returned nodes
	require.Equal(t, 2, lru.len(nil))
	for _, node := range lru.getNodes(nil) {
		if node.ID() != node1.ID() && node.ID() != node2.ID() {
			t.Fatalf("different node found %v", node)
		}
	}
}

// checks if lru is able to handle nodes with/without shards together
func TestLruMixedNodes(t *testing.T) {
	lru := newShardLRU(2)

	node1 := getEnode(t, nil, 0)
	lru.Add(node1)

	node2 := getEnode(t, nil, 1, 1)
	lru.Add(node2)
	node3 := getEnode(t, nil, 1, 2)
	lru.Add(node3)

	// check that default
	require.Equal(t, 3, lru.len(nil))

	//
	nodes := lru.getNodes(&ClusterIndex{1, 1})
	require.Equal(t, 1, len(nodes))
	require.Equal(t, node2.ID(), nodes[0].ID())

	//
	nodes = lru.getNodes(&ClusterIndex{1, 2})
	require.Equal(t, 1, len(nodes))
	require.Equal(t, node3.ID(), nodes[0].ID())
}
