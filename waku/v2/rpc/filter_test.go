package rpc

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter/pb"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

var testTopic = "test"

func makeFilterService(t *testing.T, isFullNode bool) *FilterService {
	var nodeOpts []node.WakuNodeOption

	nodeOpts = append(nodeOpts, node.WithWakuFilter(isFullNode))
	if isFullNode {
		nodeOpts = append(nodeOpts, node.WithWakuRelay())
	}

	n, err := node.New(nodeOpts...)
	require.NoError(t, err)
	err = n.Start(context.Background())
	require.NoError(t, err)

	if isFullNode {
		_, err = n.Relay().SubscribeToTopic(context.Background(), testTopic)
		require.NoError(t, err)
	}

	return NewFilterService(n, 30, utils.Logger())
}

func TestFilterSubscription(t *testing.T) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	node := relay.NewWakuRelay(host, v2.NewBroadcaster(10), 0, timesource.NewDefaultClock(), utils.Logger())
	err = node.Start(context.Background())
	require.NoError(t, err)

	_, err = node.SubscribeToTopic(context.Background(), testTopic)
	require.NoError(t, err)

	f := filter.NewWakuFilter(host, v2.NewBroadcaster(10), false, timesource.NewDefaultClock(), utils.Logger())
	err = f.Start(context.Background())
	require.NoError(t, err)

	d := makeFilterService(t, true)
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

	args := &FilterContentArgs{Topic: testTopic, ContentFilters: []*pb.FilterRequest_ContentFilter{{ContentTopic: "ct"}}}

	var reply SuccessReply
	err = d.PostV1Subscription(
		makeRequest(t),
		args,
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply)

	err = d.DeleteV1Subscription(
		makeRequest(t),
		args,
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply)
}

func TestFilterGetV1Messages(t *testing.T) {
	serviceA := makeFilterService(t, true)
	var reply SuccessReply

	serviceB := makeFilterService(t, false)
	go serviceB.Start()
	defer serviceB.Stop()

	hostInfo, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", serviceB.node.Host().ID().Pretty()))
	require.NoError(t, err)

	var addr multiaddr.Multiaddr
	for _, a := range serviceB.node.Host().Addrs() {
		addr = a.Encapsulate(hostInfo)
		break
	}
	err = serviceA.node.DialPeerWithMultiAddress(context.Background(), addr)
	require.NoError(t, err)

	// Wait for the dial to complete
	time.Sleep(1 * time.Second)

	args := &FilterContentArgs{Topic: testTopic, ContentFilters: []*pb.FilterRequest_ContentFilter{{ContentTopic: "ct"}}}
	err = serviceB.PostV1Subscription(
		makeRequest(t),
		args,
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply)

	// Wait for the subscription to be started
	time.Sleep(1 * time.Second)

	_, err = serviceA.node.Relay().PublishToTopic(
		context.Background(),
		&wpb.WakuMessage{ContentTopic: "ct"},
		testTopic,
	)
	require.NoError(t, err)
	require.True(t, reply)

	// Wait for the message to be received
	time.Sleep(1 * time.Second)

	var messagesReply1 MessagesReply
	err = serviceB.GetV1Messages(
		makeRequest(t),
		&ContentTopicArgs{"ct"},
		&messagesReply1,
	)
	require.NoError(t, err)
	require.Len(t, messagesReply1, 1)

	var messagesReply2 MessagesReply
	err = serviceB.GetV1Messages(
		makeRequest(t),
		&ContentTopicArgs{"ct"},
		&messagesReply2,
	)
	require.NoError(t, err)
	require.Len(t, messagesReply2, 0)
}
