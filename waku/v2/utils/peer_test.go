package utils

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/status-im/go-waku/tests"
	"github.com/stretchr/testify/require"
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

	proto := "test/protocol"

	h1.Peerstore().AddAddrs(h2.ID(), h2.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	h1.Peerstore().AddAddrs(h3.ID(), h2.Network().ListenAddresses(), peerstore.PermanentAddrTTL)

	// No peers with selected protocol
	_, err = SelectPeer(h1, proto)
	require.Error(t, ErrNoPeersAvailable, err)

	// Peers with selected protocol
	_ = h1.Peerstore().AddProtocols(h2.ID(), proto)
	_ = h1.Peerstore().AddProtocols(h3.ID(), proto)

	_, err = SelectPeerWithLowestRTT(ctx, h1, proto)
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

	proto := "test/protocol"

	h1.Peerstore().AddAddrs(h2.ID(), h2.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	h1.Peerstore().AddAddrs(h3.ID(), h2.Network().ListenAddresses(), peerstore.PermanentAddrTTL)

	// No peers with selected protocol
	_, err = SelectPeerWithLowestRTT(ctx, h1, proto)
	require.Error(t, ErrNoPeersAvailable, err)

	// Peers with selected protocol
	_ = h1.Peerstore().AddProtocols(h2.ID(), proto)
	_ = h1.Peerstore().AddProtocols(h3.ID(), proto)

	_, err = SelectPeerWithLowestRTT(ctx, h1, proto)
	require.NoError(t, err)
}
