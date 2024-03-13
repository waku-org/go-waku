package discv5

import (
	"context"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/service"
	"go.uber.org/zap"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/prometheus/client_golang/prometheus"

	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func discoverFilterOnDemand(iterator enode.Iterator, maxCount int) ([]service.PeerData, error) {

	log := utils.Logger()

	var peers []service.PeerData

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

		log.Info("found peer", zap.String("ID", pData.AddrInfo.ID.String()))

		if len(peers) >= maxCount {
			log.Info("found required number of nodes, stopping on demand discovery")
			break
		}

		select {
		case <-ctx.Done():
			log.Error("failed to find peers for shard and services", zap.Error(ctx.Err()))
			return nil, ctx.Err()

		default:
		}

	}

	return peers, nil
}

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

func TestDiscV5WithCapabilitiesFilter(t *testing.T) {

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
	l2, err := tests.NewLocalnode(prvKey2, ip2, udpPort2, wenr.NewWakuEnrBitfield(true, true, false, true), nil, utils.Logger())
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
	l3, err := tests.NewLocalnode(prvKey3, ip3, udpPort3, wenr.NewWakuEnrBitfield(true, true, false, true), nil, utils.Logger())
	require.NoError(t, err)
	peerconn3 := NewTestPeerDiscoverer()
	d3, err := NewDiscoveryV5(prvKey3, l3, peerconn3, prometheus.DefaultRegisterer, utils.Logger(), WithUDPPort(uint(udpPort3)), WithBootnodes([]*enode.Node{d2.localnode.Node()}))
	require.NoError(t, err)
	d3.SetHost(host3)

	defer d1.Stop()
	defer d2.Stop()
	defer d3.Stop()

	err = d1.Start(context.Background())
	require.NoError(t, err)

	err = d2.Start(context.Background())
	require.NoError(t, err)

	err = d3.Start(context.Background())
	require.NoError(t, err)

	// Desired node capabilities
	filterBitfield := wenr.NewWakuEnrBitfield(false, false, true, false)
	iterator3, err := d3.PeerIterator(FilterCapabilities(filterBitfield))
	require.NoError(t, err)
	require.NotNil(t, iterator3)

	time.Sleep(2 * time.Second)

	// Check node were discovered by automatic discovery
	require.True(t, peerconn3.HasPeer(host1.ID()) && peerconn3.HasPeer(host2.ID()))

	peers, err := discoverFilterOnDemand(iterator3, 1)
	require.NoError(t, err)
	require.Equal(t, 1, len(peers))

	// Host1 has store support while host2 hasn't
	require.Equal(t, host1.ID().String(), peers[0].AddrInfo.ID.String())

	d3.Stop()
	peerconn3.Clear()

}
