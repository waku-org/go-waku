package rpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/multiformats/go-multiaddr"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/stretchr/testify/require"
)

func makeRelayService(t *testing.T) *RelayService {
	options := node.WithWakuRelayAndMinPeers(0)
	n, err := node.New(context.Background(), options)
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)
	return NewRelayService(n)
}

func TestPostV1Message(t *testing.T) {
	var reply SuccessReply

	d := makeRelayService(t)

	err := d.PostV1Message(
		makeRequest(t),
		&RelayMessageArgs{},
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply.Success)
}

func TestRelaySubscription(t *testing.T) {
	var reply SuccessReply

	d := makeRelayService(t)
	args := &TopicsArgs{Topics: []string{"test"}}

	err := d.PostV1Subscription(
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

func TestRelayGetV1Messages(t *testing.T) {
	serviceA := makeRelayService(t)
	var reply SuccessReply

	serviceB := makeRelayService(t)
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

	args := &TopicsArgs{Topics: []string{"test"}}
	err = serviceB.PostV1Subscription(
		makeRequest(t),
		args,
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply.Success)

	// Wait for the subscription to be started
	time.Sleep(1 * time.Second)

	err = serviceA.PostV1Message(
		makeRequest(t),
		&RelayMessageArgs{
			Topic: "test",
			Message: pb.WakuMessage{
				Payload: []byte("test"),
			},
		},
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply.Success)

	// Wait for the message to be received
	time.Sleep(1 * time.Second)

	var messagesReply MessagesReply
	err = serviceB.GetV1Messages(
		makeRequest(t),
		&TopicArgs{"test"},
		&messagesReply,
	)
	require.NoError(t, err)
	require.Len(t, messagesReply.Messages, 1)

	err = serviceB.GetV1Messages(
		makeRequest(t),
		&TopicArgs{"test"},
		&messagesReply,
	)
	require.NoError(t, err)
	require.Len(t, messagesReply.Messages, 0)
}
