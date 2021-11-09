package node

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/stretchr/testify/require"
)

func TestKeepAlive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)
	host2, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	host1.Peerstore().AddAddrs(host2.ID(), host2.Addrs(), peerstore.PermanentAddrTTL)

	err = host1.Connect(ctx, host1.Peerstore().PeerInfo(host2.ID()))
	require.NoError(t, err)

	require.Len(t, host1.Network().Peers(), 1)

	ctx2, cancel2 := context.WithTimeout(ctx, 3*time.Second)
	defer cancel2()
	pingPeer(ctx2, host1, host2.ID())

	require.NoError(t, ctx.Err())
}
