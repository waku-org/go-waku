package node

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestKeepAlive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)
	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	host1.Peerstore().AddAddrs(host2.ID(), host2.Addrs(), peerstore.PermanentAddrTTL)

	err = host1.Connect(ctx, host1.Peerstore().PeerInfo(host2.ID()))
	require.NoError(t, err)

	require.Len(t, host1.Network().Peers(), 1)

	ctx2, cancel2 := context.WithTimeout(ctx, 3*time.Second)
	defer cancel2()
	wg := &sync.WaitGroup{}

	w := &WakuNode{
		host:           host1,
		wg:             wg,
		log:            utils.Logger(),
		keepAliveMutex: sync.Mutex{},
		keepAliveFails: make(map[peer.ID]int),
	}

	w.wg.Add(1)
	w.pingPeer(ctx2, w.wg, host2.ID())

	require.NoError(t, ctx.Err())
}
