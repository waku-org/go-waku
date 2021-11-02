package node

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectionStatusChanges(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connStatusChan := make(chan ConnStatus)

	// Node1: Only Relay
	hostAddr1, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	node1, err := New(ctx,
		WithHostAddress([]*net.TCPAddr{hostAddr1}),
		WithWakuRelay(),
		WithConnStatusChan(connStatusChan),
	)
	require.NoError(t, err)
	err = node1.Start()
	require.NoError(t, err)

	// Node2: Relay
	hostAddr2, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	node2, err := New(ctx,
		WithHostAddress([]*net.TCPAddr{hostAddr2}),
		WithWakuRelay(),
	)
	require.NoError(t, err)
	err = node2.Start()
	require.NoError(t, err)

	// Node3: Relay + Store
	hostAddr3, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	node3, err := New(ctx,
		WithHostAddress([]*net.TCPAddr{hostAddr3}),
		WithWakuRelay(),
		WithWakuStore(false, false),
	)
	require.NoError(t, err)
	err = node3.Start()
	require.NoError(t, err)

	err = node1.DialPeer(ctx, node2.ListenAddresses()[0].String())
	require.NoError(t, err)

	err = node1.DialPeer(ctx, node3.ListenAddresses()[0].String())
	require.NoError(t, err)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		connStatus := <-connStatusChan
		_, ok := connStatus.Peers[node2.Host().ID()]
		require.True(t, connStatus.IsOnline)
		require.True(t, ok)
		require.False(t, connStatus.HasHistory)

		connStatus = <-connStatusChan
		_, ok = connStatus.Peers[node3.Host().ID()]
		require.True(t, connStatus.IsOnline)
		require.True(t, ok)
		require.True(t, connStatus.HasHistory)
	}()

	wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()

		node3.Stop()

		connStatus := <-connStatusChan
		_, ok := connStatus.Peers[node3.Host().ID()]
		require.True(t, connStatus.IsOnline)
		require.False(t, ok)                              // Peer3 should have been disconnected
		require.False(t, connStatus.HasHistory)           // No history, because there are no peers connected with store protocol
		require.Len(t, node1.Host().Network().Peers(), 1) // No peers connected
	}()

	wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err = node1.ClosePeerById(node2.Host().ID())
		require.NoError(t, err)

		connStatus := <-connStatusChan
		_, ok := connStatus.Peers[node3.Host().ID()]
		require.False(t, connStatus.IsOnline)             // Peers are not connected. Should be offline
		require.False(t, ok)                              // Peer2 should have been disconnected
		require.False(t, connStatus.HasHistory)           // No history, because there are no peers connected with store protocol
		require.Len(t, node1.Host().Network().Peers(), 0) // No peers connected
	}()

	wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err = node1.DialPeerByID(ctx, node2.Host().ID())
		require.NoError(t, err)

		connStatus := <-connStatusChan
		_, ok := connStatus.Peers[node2.Host().ID()]
		require.True(t, connStatus.IsOnline)    // Peers2 is connected. Should be online
		require.True(t, ok)                     // Peer2 should have been connected
		require.False(t, connStatus.HasHistory) // No history because peer2 only has relay
		require.Len(t, node1.Host().Network().Peers(), 1)
	}()

	wg.Wait()
}
