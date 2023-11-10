package peermanager

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/discv5"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	wakuproto "github.com/waku-org/go-waku/waku/v2/protocol"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func getAddr(h host.Host) multiaddr.Multiaddr {
	id, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.ID().Pretty()))
	var selectedAddr multiaddr.Multiaddr
	//For now skipping circuit relay addresses as libp2p seems to be returning empty p2p-circuit addresses.
	for _, addr := range h.Network().ListenAddresses() {
		if strings.Contains(addr.String(), "p2p-circuit") {
			continue
		}
		selectedAddr = addr
	}
	return selectedAddr.Encapsulate(id)
}

func initTest(t *testing.T) (context.Context, *PeerManager, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// hosts
	h1, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)

	// host 1 is used by peer manager
	pm := NewPeerManager(10, 20, utils.Logger())
	pm.SetHost(h1)

	return ctx, pm, func() {
		cancel()
		h1.Close()
	}
}

func TestServiceSlots(t *testing.T) {
	ctx, pm, deferFn := initTest(t)
	defer deferFn()

	h2, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h2.Close()

	h3, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h3.Close()
	// protocols
	protocol := libp2pProtocol.ID("test/protocol")
	protocol1 := libp2pProtocol.ID("test/protocol1")

	// add h2 peer to peer manager
	t.Log(h2.ID())
	_, err = pm.AddPeer(getAddr(h2), wps.Static, []string{""}, false, libp2pProtocol.ID(protocol))
	require.NoError(t, err)

	///////////////
	// getting peer for protocol
	///////////////

	// select peer from pm, currently only h2 is set in pm
	peerID, err := pm.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol})
	require.NoError(t, err)
	require.Equal(t, peerID, h2.ID())

	// add h3 peer to peer manager
	_, err = pm.AddPeer(getAddr(h3), wps.Static, []string{""}, false, libp2pProtocol.ID(protocol))
	require.NoError(t, err)

	// check that returned peer is h2 or h3 peer
	peerID, err = pm.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol})
	require.NoError(t, err)
	if peerID == h2.ID() || peerID == h3.ID() {
		//Test success
		t.Log("Random peer selection per protocol successful")
	} else {
		t.FailNow()
	}

	///////////////
	// getting peer for protocol1
	///////////////
	h4, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h4.Close()

	_, err = pm.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol1})
	require.Error(t, err, ErrNoPeersAvailable)

	// add h4 peer for protocol1
	_, err = pm.AddPeer(getAddr(h4), wps.Static, []string{""}, false, libp2pProtocol.ID(protocol1))
	require.NoError(t, err)

	//Test peer selection for protocol1
	peerID, err = pm.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol1})
	require.NoError(t, err)
	require.Equal(t, peerID, h4.ID())

	_, err = pm.SelectPeerByContentTopics(protocol1, []string{""})
	require.Error(t, wakuproto.ErrInvalidFormat, err)

}

func TestPeerSelection(t *testing.T) {
	ctx, pm, deferFn := initTest(t)
	defer deferFn()

	h2, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h2.Close()

	h3, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h3.Close()

	protocol := libp2pProtocol.ID("test/protocol")
	_, err = pm.AddPeer(getAddr(h2), wps.Static, []string{"/waku/2/rs/2/1", "/waku/2/rs/2/2"}, false, libp2pProtocol.ID(protocol))
	require.NoError(t, err)

	_, err = pm.AddPeer(getAddr(h3), wps.Static, []string{"/waku/2/rs/2/1"}, false, libp2pProtocol.ID(protocol))
	require.NoError(t, err)

	_, err = pm.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol})
	require.NoError(t, err)

	peerID, err := pm.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol, PubsubTopics: []string{"/waku/2/rs/2/2"}})
	require.NoError(t, err)
	require.Equal(t, h2.ID(), peerID)

	_, err = pm.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol, PubsubTopics: []string{"/waku/2/rs/2/3"}})
	require.Error(t, ErrNoPeersAvailable, err)

	_, err = pm.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol, PubsubTopics: []string{"/waku/2/rs/2/1"}})
	require.NoError(t, err)

	//Test for selectWithLowestRTT
	_, err = pm.SelectPeer(PeerSelectionCriteria{SelectionType: LowestRTT, Proto: protocol, PubsubTopics: []string{"/waku/2/rs/2/1"}})
	require.NoError(t, err)
}

func TestDefaultProtocol(t *testing.T) {
	ctx, pm, deferFn := initTest(t)
	defer deferFn()
	///////////////
	// check peer for default protocol
	///////////////
	//Test empty peer selection for relay protocol
	_, err := pm.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, Proto: relay.WakuRelayID_v200})
	require.Error(t, err, ErrNoPeersAvailable)

	///////////////
	// getting peer for default protocol
	///////////////
	h5, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h5.Close()

	//Test peer selection for relay protocol from peer store
	_, err = pm.AddPeer(getAddr(h5), wps.Static, []string{""}, false, relay.WakuRelayID_v200)
	require.NoError(t, err)

	// since we are not passing peerList, selectPeer fn using filterByProto checks in PeerStore for peers with same protocol.
	peerID, err := pm.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, Proto: relay.WakuRelayID_v200})
	require.NoError(t, err)
	require.Equal(t, peerID, h5.ID())
}

func TestAdditionAndRemovalOfPeer(t *testing.T) {
	ctx, pm, deferFn := initTest(t)
	defer deferFn()
	///////////////
	// set h6 peer for protocol2 and remove that peer and check again
	///////////////
	//Test random peer selection
	protocol2 := libp2pProtocol.ID("test/protocol2")
	h6, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h6.Close()

	_, err = pm.AddPeer(getAddr(h6), wps.Static, []string{""}, false, protocol2)
	require.NoError(t, err)

	peerID, err := pm.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol2})
	require.NoError(t, err)
	require.Equal(t, peerID, h6.ID())

	pm.RemovePeer(peerID)
	_, err = pm.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol2})
	require.Error(t, err, ErrNoPeersAvailable)
}

func TestConnectToRelayPeers(t *testing.T) {

	ctx, pm, deferFn := initTest(t)
	pc, err := NewPeerConnectionStrategy(pm, 120*time.Second, pm.logger)
	require.NoError(t, err)
	err = pc.Start(ctx)
	require.NoError(t, err)
	pm.Start(ctx)

	defer deferFn()

	pm.connectToRelayPeers()

}

func createHostWithDiscv5AndPM(t *testing.T, hostName string, topic string, enrField uint8, bootnode ...*enode.Node) (host.Host, *PeerManager, *discv5.DiscoveryV5) {
	ps, err := pstoremem.NewPeerstore()
	require.NoError(t, err)
	wakuPeerStore := wps.NewWakuPeerstore(ps)

	host, _, prvKey1 := tests.CreateHost(t, libp2p.Peerstore(wakuPeerStore))

	logger := utils.Logger().Named(hostName)

	udpPort, err := tests.FindFreeUDPPort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	ip1, _ := tests.ExtractIP(host.Addrs()[0])
	localNode, err := tests.NewLocalnode(prvKey1, ip1, udpPort, enrField, nil, logger)
	require.NoError(t, err)

	rs, err := wakuproto.TopicsToRelayShards(topic)
	require.NoError(t, err)

	err = wenr.Update(localNode, wenr.WithWakuRelaySharding(rs[0]))
	require.NoError(t, err)
	pm := NewPeerManager(10, 20, logger)
	pm.SetHost(host)
	peerconn, err := NewPeerConnectionStrategy(pm, 30*time.Second, logger)
	require.NoError(t, err)
	discv5, err := discv5.NewDiscoveryV5(prvKey1, localNode, peerconn, prometheus.DefaultRegisterer, logger, discv5.WithUDPPort(uint(udpPort)), discv5.WithBootnodes(bootnode))
	require.NoError(t, err)
	discv5.SetHost(host)
	pm.SetDiscv5(discv5)
	pm.SetPeerConnector(peerconn)

	return host, pm, discv5
}

func TestOnDemandPeerDiscovery(t *testing.T) {
	topic := "/waku/2/rs/1/1"

	// Host1 <-> Host2 <-> Host3
	host1, _, d1 := createHostWithDiscv5AndPM(t, "host1", topic, wenr.NewWakuEnrBitfield(true, true, false, true))

	host2, _, d2 := createHostWithDiscv5AndPM(t, "host2", topic, wenr.NewWakuEnrBitfield(false, true, true, true), d1.Node())
	host3, pm3, d3 := createHostWithDiscv5AndPM(t, "host3", topic, wenr.NewWakuEnrBitfield(true, true, true, true), d2.Node())

	defer d1.Stop()
	defer d2.Stop()
	defer d3.Stop()

	defer host1.Close()
	defer host2.Close()
	defer host3.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := d1.Start(ctx)
	require.NoError(t, err)

	err = d2.Start(ctx)
	require.NoError(t, err)

	err = d3.Start(ctx)
	require.NoError(t, err)

	//Discovery should fail for non-waku protocol
	_, err = pm3.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, PubsubTopics: []string{topic}, Proto: "/test"})
	require.Error(t, err)

	_, err = pm3.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, Proto: "/test"})
	require.Error(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var enrField uint8
	enrField |= (1 << 1)
	pm3.RegisterWakuProtocol("/vac/waku/store/2.0.0-beta4", enrField)
	peerID, err := pm3.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, PubsubTopics: []string{topic}, Proto: "/vac/waku/store/2.0.0-beta4", Ctx: ctx})
	require.NoError(t, err)
	require.Equal(t, peerID, host2.ID())

	var enrField1 uint8

	enrField1 |= (1 << 3)
	pm3.RegisterWakuProtocol("/vac/waku/lightpush/2.0.0-beta1", enrField1)
	peerID, err = pm3.SelectPeer(PeerSelectionCriteria{SelectionType: Automatic, PubsubTopics: []string{topic}, Proto: "/vac/waku/lightpush/2.0.0-beta1", Ctx: ctx})
	require.NoError(t, err)
	require.Equal(t, peerID, host1.ID())

}
