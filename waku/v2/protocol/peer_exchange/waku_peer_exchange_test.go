package peer_exchange

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/discv5"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestRetrieveProvidePeerExchangePeers(t *testing.T) {
	// H1
	host1, _, prvKey1 := tests.CreateHost(t)
	udpPort1, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	ip1, _ := tests.ExtractIP(host1.Addrs()[0])
	l1, err := tests.NewLocalnode(prvKey1, ip1, udpPort1, wenr.NewWakuEnrBitfield(false, false, false, true), nil, utils.Logger())
	require.NoError(t, err)
	discv5PeerConn1 := discv5.NewTestPeerDiscoverer()
	d1, err := discv5.NewDiscoveryV5(prvKey1, l1, discv5PeerConn1, prometheus.DefaultRegisterer, utils.Logger(), discv5.WithUDPPort(uint(udpPort1)))
	require.NoError(t, err)
	d1.SetHost(host1)

	// H2
	host2, _, prvKey2 := tests.CreateHost(t)
	ip2, _ := tests.ExtractIP(host2.Addrs()[0])
	udpPort2, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	l2, err := tests.NewLocalnode(prvKey2, ip2, udpPort2, wenr.NewWakuEnrBitfield(false, false, false, true), nil, utils.Logger())
	require.NoError(t, err)
	discv5PeerConn2 := discv5.NewTestPeerDiscoverer()
	d2, err := discv5.NewDiscoveryV5(prvKey2, l2, discv5PeerConn2, prometheus.DefaultRegisterer, utils.Logger(), discv5.WithUDPPort(uint(udpPort2)), discv5.WithBootnodes([]*enode.Node{d1.Node()}))
	require.NoError(t, err)
	d2.SetHost(host2)

	// H3
	host3, _, _ := tests.CreateHost(t)

	defer d1.Stop()
	defer d2.Stop()
	defer host1.Close()
	defer host2.Close()
	defer host3.Close()

	err = d1.Start(context.Background())
	require.NoError(t, err)

	err = d2.Start(context.Background())
	require.NoError(t, err)

	// mount peer exchange
	pxPeerConn1 := discv5.NewTestPeerDiscoverer()
	px1, err := NewWakuPeerExchange(d1, pxPeerConn1, nil, prometheus.DefaultRegisterer, utils.Logger())
	require.NoError(t, err)
	px1.SetHost(host1)

	pxPeerConn3 := discv5.NewTestPeerDiscoverer()
	px3, err := NewWakuPeerExchange(nil, pxPeerConn3, nil, prometheus.DefaultRegisterer, utils.Logger())
	require.NoError(t, err)
	px3.SetHost(host3)

	err = px1.Start(context.Background())
	require.NoError(t, err)

	err = px3.Start(context.Background())
	require.NoError(t, err)

	host3.Peerstore().AddAddrs(host1.ID(), host1.Addrs(), peerstore.PermanentAddrTTL)
	err = host3.Peerstore().AddProtocols(host1.ID(), PeerExchangeID_v20alpha1)
	require.NoError(t, err)

	time.Sleep(3 * time.Second) // Wait some time for peers to be discovered

	err = px3.Request(context.Background(), 1, WithPeer(host1.ID()))
	require.NoError(t, err)

	time.Sleep(3 * time.Second) //  Give the algorithm some time to work its magic

	require.True(t, pxPeerConn3.HasPeer(host2.ID()))

	px1.Stop()
	px3.Stop()
}
