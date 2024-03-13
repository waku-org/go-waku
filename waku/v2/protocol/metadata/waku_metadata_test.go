package metadata

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peerstore"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multistream"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func createWakuMetadata(t *testing.T, rs *protocol.RelayShards) *WakuMetadata {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)
	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	key, _ := gcrypto.GenerateKey()

	localNode, err := enr.NewLocalnode(key)
	require.NoError(t, err)

	clusterID := uint16(0)
	if rs != nil {
		err = enr.WithWakuRelaySharding(*rs)(localNode)
		require.NoError(t, err)
		clusterID = rs.ClusterID
	}

	m1 := NewWakuMetadata(clusterID, localNode, utils.Logger())
	m1.SetHost(host)
	err = m1.Start(context.TODO())
	require.NoError(t, err)

	return m1
}

func isProtocolNotSupported(err error) bool {
	notSupportedErr := multistream.ErrNotSupported[libp2pProtocol.ID]{}
	return errors.Is(err, notSupportedErr)
}

func TestWakuMetadataRequest(t *testing.T) {
	testShard16 := uint16(16)

	rs16_1, err := protocol.NewRelayShards(testShard16, 1)
	require.NoError(t, err)
	rs16_2, err := protocol.NewRelayShards(testShard16, 2)
	require.NoError(t, err)

	m16_1 := createWakuMetadata(t, &rs16_1)
	m16_2 := createWakuMetadata(t, &rs16_2)
	m_noRS := createWakuMetadata(t, nil)

	m16_1.h.Peerstore().AddAddrs(m16_2.h.ID(), m16_2.h.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	m16_1.h.Peerstore().AddAddrs(m_noRS.h.ID(), m_noRS.h.Network().ListenAddresses(), peerstore.PermanentAddrTTL)

	// Query a peer that is subscribed to a shard
	result, err := m16_1.Request(context.Background(), m16_2.h.ID())
	require.NoError(t, err)
	require.Equal(t, testShard16, result.ClusterID)
	require.Equal(t, rs16_2.ShardIDs, result.ShardIDs)

	// Updating the peer shards
	rs16_2.ShardIDs = append(rs16_2.ShardIDs, 3, 4)
	err = enr.WithWakuRelaySharding(rs16_2)(m16_2.localnode)
	require.NoError(t, err)

	// Query same peer, after that peer subscribes to more shards
	result, err = m16_1.Request(context.Background(), m16_2.h.ID())
	require.NoError(t, err)
	require.Equal(t, testShard16, result.ClusterID)
	require.ElementsMatch(t, rs16_2.ShardIDs, result.ShardIDs)

	// Query a peer not subscribed to any shard
	_, err = m16_1.Request(context.Background(), m_noRS.h.ID())
	require.True(t, isProtocolNotSupported(err))
}
