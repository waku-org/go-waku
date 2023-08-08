package tests

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestSelectPeer(t *testing.T) {
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

	proto := protocol.ID("test/protocol")

	h1.Peerstore().AddAddrs(h2.ID(), h2.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	h1.Peerstore().AddAddrs(h3.ID(), h3.Network().ListenAddresses(), peerstore.PermanentAddrTTL)

	// No peers with selected protocol
	_, err = utils.SelectPeer(h1, proto, nil, utils.Logger())
	require.Error(t, utils.ErrNoPeersAvailable, err)

	// Peers with selected protocol
	_ = h1.Peerstore().AddProtocols(h2.ID(), proto)
	_ = h1.Peerstore().AddProtocols(h3.ID(), proto)

	_, err = utils.SelectPeerWithLowestRTT(ctx, h1, proto, nil, utils.Logger())
	require.NoError(t, err)

}

func TestSelectPeerWithLowestRTT(t *testing.T) {
	// help-wanted: how to slowdown the ping response to properly test the rtt

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

	proto := protocol.ID("test/protocol")

	h1.Peerstore().AddAddrs(h2.ID(), h2.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	h1.Peerstore().AddAddrs(h3.ID(), h2.Network().ListenAddresses(), peerstore.PermanentAddrTTL)

	// No peers with selected protocol
	_, err = utils.SelectPeerWithLowestRTT(ctx, h1, proto, nil, utils.Logger())
	require.Error(t, utils.ErrNoPeersAvailable, err)

	// Peers with selected protocol
	_ = h1.Peerstore().AddProtocols(h2.ID(), proto)
	_ = h1.Peerstore().AddProtocols(h3.ID(), proto)

	_, err = utils.SelectPeerWithLowestRTT(ctx, h1, proto, nil, utils.Logger())
	require.NoError(t, err)
}
