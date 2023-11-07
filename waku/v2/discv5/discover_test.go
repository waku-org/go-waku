package discv5

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/prometheus/client_golang/prometheus"

	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestDiscV5(t *testing.T) {
	// Host1 <-> Host2 <-> Host3
	// Host4(No waku capabilities) <-> Host2

	// H1
	host1, _, prvKey1 := tests.CreateHost(t)
	udpPort1, err := tests.FindFreeUDPPort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	ip1, _ := tests.ExtractIP(host1.Addrs()[0])
	l1, err := tests.NewLocalnode(prvKey1, ip1, udpPort1, wenr.NewWakuEnrBitfield(true, true, true, true), nil, utils.Logger())
	require.NoError(t, err)
	peerconn1 := NewTestPeerDiscoverer()
	d1, err := NewDiscoveryV5(prvKey1, l1, peerconn1, prometheus.DefaultRegisterer, utils.Logger(), WithUDPPort(uint(udpPort1)))
	require.NoError(t, err)
	d1.SetHost(host1)

	// H2
	host2, _, prvKey2 := tests.CreateHost(t)
	ip2, _ := tests.ExtractIP(host2.Addrs()[0])
	udpPort2, err := tests.FindFreeUDPPort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	l2, err := tests.NewLocalnode(prvKey2, ip2, udpPort2, wenr.NewWakuEnrBitfield(true, true, true, true), nil, utils.Logger())
	require.NoError(t, err)
	peerconn2 := NewTestPeerDiscoverer()
	d2, err := NewDiscoveryV5(prvKey2, l2, peerconn2, prometheus.DefaultRegisterer, utils.Logger(), WithUDPPort(uint(udpPort2)), WithBootnodes([]*enode.Node{d1.localnode.Node()}))
	require.NoError(t, err)
	d2.SetHost(host2)

	// H3
	host3, _, prvKey3 := tests.CreateHost(t)
	ip3, _ := tests.ExtractIP(host3.Addrs()[0])
	udpPort3, err := tests.FindFreeUDPPort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	l3, err := tests.NewLocalnode(prvKey3, ip3, udpPort3, wenr.NewWakuEnrBitfield(true, true, true, true), nil, utils.Logger())
	require.NoError(t, err)
	peerconn3 := NewTestPeerDiscoverer()
	d3, err := NewDiscoveryV5(prvKey3, l3, peerconn3, prometheus.DefaultRegisterer, utils.Logger(), WithUDPPort(uint(udpPort3)), WithBootnodes([]*enode.Node{d2.localnode.Node()}))
	require.NoError(t, err)
	d3.SetHost(host3)

	// H4 doesn't have any Waku capabilities
	host4, _, prvKey4 := tests.CreateHost(t)
	ip4, _ := tests.ExtractIP(host2.Addrs()[0])
	udpPort4, err := tests.FindFreeUDPPort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	l4, err := tests.NewLocalnode(prvKey4, ip4, udpPort4, 0, nil, utils.Logger())
	require.NoError(t, err)
	peerconn4 := NewTestPeerDiscoverer()
	d4, err := NewDiscoveryV5(prvKey4, l4, peerconn4, prometheus.DefaultRegisterer, utils.Logger(), WithUDPPort(uint(udpPort4)), WithBootnodes([]*enode.Node{d2.localnode.Node()}))
	require.NoError(t, err)
	d2.SetHost(host2)

	defer d1.Stop()
	defer d2.Stop()
	defer d3.Stop()
	defer d4.Stop()

	err = d1.Start(context.Background())
	require.NoError(t, err)

	err = d2.Start(context.Background())
	require.NoError(t, err)

	err = d3.Start(context.Background())
	require.NoError(t, err)

	err = d4.Start(context.Background())
	require.NoError(t, err)

	time.Sleep(2 * time.Second) // Wait for nodes to be discovered

	require.True(t, peerconn3.HasPeer(host1.ID()) && peerconn3.HasPeer(host2.ID()))
	require.False(t, peerconn3.HasPeer(host4.ID())) //host4 should not be discoverable, rather filtered out.

	d3.Stop()
	peerconn3.Clear()

	// Restart peer search
	err = d3.Start(context.Background())
	require.NoError(t, err)

	time.Sleep(2 * time.Second) // Wait for nodes to be discovered

	require.True(t, peerconn3.HasPeer(host1.ID()) && peerconn3.HasPeer(host2.ID()))
	require.False(t, peerconn3.HasPeer(host4.ID())) //host4 should not be discoverable, rather filtered out.

}
