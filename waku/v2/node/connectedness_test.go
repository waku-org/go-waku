package node

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/persistence/sqlite"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func goCheckConnectedness(t *testing.T, wg *sync.WaitGroup, connStatusChan chan ConnStatus, clientNode *WakuNode, node *WakuNode, nodeShouldBeConnected bool, shouldBeOnline bool, shouldHaveHistory bool, expectedPeers int) {
	wg.Add(1)
	go checkConnectedness(t, wg, connStatusChan, clientNode, node, nodeShouldBeConnected, shouldBeOnline, shouldHaveHistory, expectedPeers)
}

func checkConnectedness(t *testing.T, wg *sync.WaitGroup, connStatusChan chan ConnStatus, clientNode *WakuNode, node *WakuNode, nodeShouldBeConnected bool, shouldBeOnline bool, shouldHaveHistory bool, expectedPeers int) {
	defer wg.Done()

	timeout := time.After(5 * time.Second)

	select {
	case connStatus := <-connStatusChan:
		_, ok := connStatus.Peers[node.Host().ID()]
		if (nodeShouldBeConnected && ok) || (!nodeShouldBeConnected && !ok) {
			// Only execute the test when the node is connected or disconnected and it does not appear in the map returned by the connection status channel
			require.True(t, connStatus.IsOnline == shouldBeOnline)
			require.True(t, connStatus.HasHistory == shouldHaveHistory)
			require.Len(t, clientNode.Host().Network().Peers(), expectedPeers)
			return
		}

	case <-timeout:
		require.Fail(t, "node should have connected")

	}
}

func TestConnectionStatusChanges(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connStatusChan := make(chan ConnStatus, 100)

	// Node1: Only Relay
	hostAddr1, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	node1, err := New(
		WithHostAddress(hostAddr1),
		WithWakuRelay(),
		WithConnectionStatusChannel(connStatusChan),
	)
	require.NoError(t, err)
	err = node1.Start(ctx)
	require.NoError(t, err)

	// Node2: Relay
	hostAddr2, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	node2, err := New(
		WithHostAddress(hostAddr2),
		WithWakuRelay(),
	)
	require.NoError(t, err)
	err = node2.Start(ctx)
	require.NoError(t, err)

	db, err := sqlite.NewDB(":memory:", false, utils.Logger())
	require.NoError(t, err)
	dbStore, err := persistence.NewDBStore(prometheus.DefaultRegisterer, utils.Logger(), persistence.WithDB(db), persistence.WithMigrations(sqlite.Migrations))
	require.NoError(t, err)

	// Node3: Relay + Store
	hostAddr3, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	node3, err := New(
		WithHostAddress(hostAddr3),
		WithWakuRelay(),
		WithWakuStore(),
		WithMessageProvider(dbStore),
	)
	require.NoError(t, err)
	err = node3.Start(ctx)
	require.NoError(t, err)

	var wg sync.WaitGroup

	goCheckConnectedness(t, &wg, connStatusChan, node1, node2, true, true, false, 1)

	err = node1.DialPeer(ctx, node2.ListenAddresses()[0].String())
	require.NoError(t, err)

	wg.Wait()

	goCheckConnectedness(t, &wg, connStatusChan, node1, node3, true, true, true, 2)

	err = node1.DialPeer(ctx, node3.ListenAddresses()[0].String())
	require.NoError(t, err)

	goCheckConnectedness(t, &wg, connStatusChan, node1, node3, false, true, false, 1)

	node3.Stop()

	wg.Wait()

	goCheckConnectedness(t, &wg, connStatusChan, node1, node2, false, false, false, 0)

	err = node1.ClosePeerById(node2.Host().ID())
	require.NoError(t, err)
	wg.Wait()

	goCheckConnectedness(t, &wg, connStatusChan, node1, node2, true, true, false, 1)

	err = node1.DialPeerByID(ctx, node2.Host().ID())
	require.NoError(t, err)
	wg.Wait()
}
