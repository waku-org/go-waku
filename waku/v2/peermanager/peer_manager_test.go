package peermanager

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func getAddr(h host.Host) multiaddr.Multiaddr {
	id, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.ID().Pretty()))
	return h.Network().ListenAddresses()[0].Encapsulate(id)
}

func initTest(t *testing.T) (context.Context, *PeerManager, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// hosts
	h1, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h1.Close()

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
	_, err = pm.AddPeer(getAddr(h2), wps.Static, []string{""}, libp2pProtocol.ID(protocol))
	require.NoError(t, err)

	///////////////
	// getting peer for protocol
	///////////////

	// select peer from pm, currently only h2 is set in pm
	peerID, err := pm.SelectPeer(protocol, nil)
	require.NoError(t, err)
	require.Equal(t, peerID, h2.ID())

	// add h3 peer to peer manager
	_, err = pm.AddPeer(getAddr(h3), wps.Static, []string{""}, libp2pProtocol.ID(protocol))
	require.NoError(t, err)

	// check that returned peer is h2 or h3 peer
	peerID, err = pm.SelectPeer(protocol, nil)
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

	_, err = pm.SelectPeer(protocol1, nil)
	require.Error(t, err, utils.ErrNoPeersAvailable)

	// add h4 peer for protocol1
	_, err = pm.AddPeer(getAddr(h4), wps.Static, []string{""}, libp2pProtocol.ID(protocol1))
	require.NoError(t, err)

	//Test peer selection for protocol1
	peerID, err = pm.SelectPeer(protocol1, nil)
	require.NoError(t, err)
	require.Equal(t, peerID, h4.ID())

}

func TestDefaultProtocol(t *testing.T) {
	ctx, pm, deferFn := initTest(t)
	defer deferFn()
	///////////////
	// check peer for default protocol
	///////////////
	//Test empty peer selection for relay protocol
	_, err := pm.SelectPeer(relay.WakuRelayID_v200, nil)
	require.Error(t, err, utils.ErrNoPeersAvailable)

	///////////////
	// getting peer for default protocol
	///////////////
	h5, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h5.Close()

	//Test peer selection for relay protocol from peer store
	_, err = pm.AddPeer(getAddr(h5), wps.Static, []string{""}, relay.WakuRelayID_v200)
	require.NoError(t, err)

	// since we are not passing peerList, selectPeer fn using filterByProto checks in PeerStore for peers with same protocol.
	peerID, err := pm.SelectPeer(relay.WakuRelayID_v200, nil)
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

	_, err = pm.AddPeer(getAddr(h6), wps.Static, []string{""}, protocol2)
	require.NoError(t, err)

	peerID, err := pm.SelectPeer(protocol2, nil)
	require.NoError(t, err)
	require.Equal(t, peerID, h6.ID())

	pm.RemovePeer(peerID)
	_, err = pm.SelectPeer(protocol2, nil)
	require.Error(t, err, utils.ErrNoPeersAvailable)
}

func TestConnectToRelayPeers(t *testing.T) {

	ctx, pm, deferFn := initTest(t)
	pc, err := NewPeerConnectionStrategy(pm, 120*time.Second, pm.logger)
	require.NoError(t, err)
	pm.SetPeerConnector(pc)
	err = pc.Start(ctx)
	require.NoError(t, err)
	pm.Start(ctx)

	defer deferFn()

	pm.connectToRelayPeers()

}
