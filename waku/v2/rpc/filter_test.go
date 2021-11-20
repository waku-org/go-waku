package rpc

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/multiformats/go-multiaddr"
	"github.com/status-im/go-waku/tests"
	v2 "github.com/status-im/go-waku/waku/v2"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/stretchr/testify/require"
)

var testTopic = "test"

func makeFilterService(t *testing.T) *FilterService {
	n, err := node.New(context.Background(), node.WithWakuFilter(true), node.WithWakuRelay())
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)

	_, err = n.Relay().SubscribeToTopic(context.Background(), testTopic)
	require.NoError(t, err)

	return &FilterService{n}
}

func TestFilterSubscription(t *testing.T) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	node, err := relay.NewWakuRelay(context.Background(), host, v2.NewBroadcaster(10))
	require.NoError(t, err)

	_, err = node.SubscribeToTopic(context.Background(), testTopic)
	require.NoError(t, err)

	_ = filter.NewWakuFilter(context.Background(), host, false)

	d := makeFilterService(t)
	defer d.node.Stop()

	hostInfo, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", host.ID().Pretty()))
	require.NoError(t, err)

	var addr multiaddr.Multiaddr
	for _, a := range host.Addrs() {
		addr = a.Encapsulate(hostInfo)
		break
	}

	_, err = d.node.AddPeer(addr, filter.FilterID_v20beta1)
	require.NoError(t, err)

	args := &FilterContentFilterArgs{Topic: testTopic, ContentFilters: []pb.ContentFilter{{ContentTopic: "ct"}}}

	var reply SuccessReply
	err = d.PostV1Subscription(
		makeRequest(t),
		args,
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply.Success)

	err = d.DeleteV1Subscription(
		makeRequest(t),
		args,
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply.Success)
}
