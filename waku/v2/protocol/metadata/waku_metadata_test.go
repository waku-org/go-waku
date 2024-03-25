package metadata

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
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

func isStreamReset(err error) bool {
	return strings.Contains(err.Error(), "stream reset")
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
	require.True(t, isProtocolNotSupported(err) || isStreamReset(err))
}

func TestNoNetwork(t *testing.T) {
	cluster1 := uint16(1)

	rs1, err := protocol.NewRelayShards(cluster1, 1)
	require.NoError(t, err)
	m1 := createWakuMetadata(t, &rs1)

	// host2 does not support metadata protocol, so it should be dropped
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)
	host2, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	m1.h.Peerstore().AddAddrs(host2.ID(), host2.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	_, err = m1.h.Network().DialPeer(context.TODO(), host2.ID())
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// Verifying peer connections
	require.Len(t, m1.h.Network().Peers(), 0)
	require.Len(t, host2.Network().Peers(), 0)
}

func TestDropConnectionOnDiffNetworks(t *testing.T) {
	cluster1 := uint16(1)
	cluster2 := uint16(2)

	// Initializing metadata and peer managers

	rs1, err := protocol.NewRelayShards(cluster1, 1)
	require.NoError(t, err)
	m1 := createWakuMetadata(t, &rs1)

	rs2, err := protocol.NewRelayShards(cluster2, 1)
	require.NoError(t, err)
	m2 := createWakuMetadata(t, &rs2)

	rs3, err := protocol.NewRelayShards(cluster2, 1)
	require.NoError(t, err)
	m3 := createWakuMetadata(t, &rs3)

	// Creating connection between peers

	// 1->2 (fails)
	m1.h.Peerstore().AddAddrs(m2.h.ID(), m2.h.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	_, err = m1.h.Network().DialPeer(context.TODO(), m2.h.ID())
	require.NoError(t, err)

	// 1->3 (fails)
	m1.h.Peerstore().AddAddrs(m3.h.ID(), m3.h.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	_, err = m1.h.Network().DialPeer(context.TODO(), m3.h.ID())
	require.NoError(t, err)

	// 2->3 (succeeds)
	m2.h.Peerstore().AddAddrs(m3.h.ID(), m3.h.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	_, err = m2.h.Network().DialPeer(context.TODO(), m3.h.ID())
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// Verifying peer connections
	require.Len(t, m1.h.Network().Peers(), 0)
	require.Len(t, m2.h.Network().Peers(), 1)
	require.Len(t, m3.h.Network().Peers(), 1)
	require.Equal(t, []peer.ID{m3.h.ID()}, m2.h.Network().Peers())
	require.Equal(t, []peer.ID{m2.h.ID()}, m3.h.Network().Peers())

}
