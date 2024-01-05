package peermanager

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestRTT(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	h1, _ := tests.MakeHost(ctx, 0, rand.Reader)
	h2, _ := tests.MakeHost(ctx, 0, rand.Reader)
	h3, _ := tests.MakeHost(ctx, 0, rand.Reader)

	h1.Peerstore().AddAddrs(h2.ID(), h2.Addrs(), peerstore.PermanentAddrTTL)
	h1.Peerstore().AddAddrs(h3.ID(), h3.Addrs(), peerstore.PermanentAddrTTL)

	rtt := NewRTTCache(timesource.NewDefaultClock(), utils.Logger())
	rtt.SetHost(h1)
	go rtt.start(ctx, 500*time.Millisecond)

	time.Sleep(2 * time.Second)

	rtt.RLock()
	_, exists := rtt.peers[h2.ID()]
	require.True(t, exists)
	_, exists = rtt.peers[h3.ID()]
	require.True(t, exists)
	rtt.RUnlock()

	// Simulating h3 not being available
	h3.Close()
	time.Sleep(2 * time.Second)

	rtt.RLock()
	h3RTT := rtt.peers[h3.ID()]
	require.Equal(t, h3RTT.PingRTT, time.Hour) // Should have 1hr

	// Determine next verification ordering is correct
	currN := rtt.pingQueue[0].nextVerification
	for _, p := range rtt.pingQueue {
		if currN.Before(p.nextVerification) {
			t.Error("invalid ordering", currN, p.PingRTT)
		}
		currN = p.nextVerification
	}
	rtt.RUnlock()

	fastestPeer, err := rtt.FastestPeer(ctx, peer.IDSlice{h2.ID(), h3.ID()})
	require.NoError(t, err)
	require.Equal(t, h2.ID(), fastestPeer.PeerID)
}
