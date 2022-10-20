package rpc

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/multiformats/go-multiaddr"
	"github.com/status-im/go-waku/tests"
	v2 "github.com/status-im/go-waku/waku/v2"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/status-im/go-waku/waku/v2/utils"
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

	return NewFilterService(n, 30, utils.Logger())
}

func TestFilterSubscription(t *testing.T) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	node, err := relay.NewWakuRelay(context.Background(), host, v2.NewBroadcaster(10), 0, utils.Logger())
	require.NoError(t, err)

	_, err = node.SubscribeToTopic(context.Background(), testTopic)
	require.NoError(t, err)

	_, _ = filter.NewWakuFilter(context.Background(), host, false, utils.Logger())

	d := makeFilterService(t)
	defer d.node.Stop()

	hostInfo, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", host.ID().Pretty()))
	require.NoError(t, err)

	var addr multiaddr.Multiaddr
	for _, a := range host.Addrs() {
		addr = a.Encapsulate(hostInfo)
		break
	}

	_, err = d.node.AddPeer(addr, string(filter.FilterID_v20beta1))
	require.NoError(t, err)

	args := &FilterContentArgs{Topic: testTopic, ContentFilters: []pb.ContentFilter{{ContentTopic: "ct"}}}

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
	serviceA := makeFilterService(t)
	var reply SuccessReply

	serviceB := makeFilterService(t)
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

	args := &FilterContentArgs{Topic: testTopic, ContentFilters: []pb.ContentFilter{{ContentTopic: "ct"}}}
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
		&pb.WakuMessage{ContentTopic: "ct"},
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
