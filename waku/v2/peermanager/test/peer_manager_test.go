package test

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peerstore"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestServiceSlots(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	h1, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h1.Close()

	h2, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h2.Close()

	h3, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h3.Close()
	protocol := libp2pProtocol.ID("test/protocol")
	protocol1 := libp2pProtocol.ID("test/protocol1")

	pm := peermanager.NewPeerManager(10, utils.Logger())
	pm.SetHost(h1)

	h1.Peerstore().AddAddrs(h2.ID(), h2.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	err = h1.Peerstore().AddProtocols(h2.ID(), libp2pProtocol.ID(protocol))
	require.NoError(t, err)

	//Test selection from peerStore.
	peerId, err := pm.SelectPeer(protocol, nil, utils.Logger())
	require.NoError(t, err)
	require.Equal(t, peerId, h2.ID())

	//Test addition and selection from service-slot
	pm.AddPeerToServiceSlot(protocol, h2.ID())

	peerId, err = pm.SelectPeer(protocol, nil, utils.Logger())
	require.NoError(t, err)
	require.Equal(t, peerId, h2.ID())

	h1.Peerstore().AddAddrs(h3.ID(), h3.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	pm.AddPeerToServiceSlot(protocol, h3.ID())

	h4, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h4.Close()

	h1.Peerstore().AddAddrs(h4.ID(), h4.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	pm.AddPeerToServiceSlot(protocol1, h4.ID())

	//Test peer selection from recently added peer to serviceSlot
	peerId, err = pm.SelectPeer(protocol, nil, utils.Logger())
	require.NoError(t, err)
	require.Equal(t, peerId, h3.ID())

	//Test peer selection for specific protocol
	peerId, err = pm.SelectPeer(protocol1, nil, utils.Logger())
	require.NoError(t, err)
	require.Equal(t, peerId, h4.ID())

	h5, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h5.Close()

	//Test empty peer selection for relay protocol
	_, err = pm.SelectPeer(peermanager.WakuRelayIDv200, nil, utils.Logger())
	require.Error(t, err, utils.ErrNoPeersAvailable)
	//Test peer selection for relay protocol from peer store
	h1.Peerstore().AddAddrs(h5.ID(), h5.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	pm.AddPeerToServiceSlot(peermanager.WakuRelayIDv200, h5.ID())

	_, err = pm.SelectPeer(peermanager.WakuRelayIDv200, nil, utils.Logger())
	require.Error(t, err, utils.ErrNoPeersAvailable)

	err = h1.Peerstore().AddProtocols(h5.ID(), peermanager.WakuRelayIDv200)
	require.NoError(t, err)

	peerId, err = pm.SelectPeer(peermanager.WakuRelayIDv200, nil, utils.Logger())
	require.NoError(t, err)
	require.Equal(t, peerId, h5.ID())

	//Test random peer selection
	protocol2 := libp2pProtocol.ID("test/protocol2")
	h6, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h6.Close()

	h1.Peerstore().AddAddrs(h6.ID(), h6.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	err = h1.Peerstore().AddProtocols(h6.ID(), libp2pProtocol.ID(protocol2))
	require.NoError(t, err)

	peerId, err = pm.SelectPeer(protocol2, nil, utils.Logger())
	require.NoError(t, err)
	require.Equal(t, peerId, h6.ID())

	pm.RemovePeer(peerId)
	_, err = pm.SelectPeer(protocol2, nil, utils.Logger())
	require.Error(t, err, utils.ErrNoPeersAvailable)
}
