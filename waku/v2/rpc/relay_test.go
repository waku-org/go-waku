package rpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func makeRelayService(t *testing.T) *RelayService {
	options := node.WithWakuRelayAndMinPeers(0)
	n, err := node.New(context.Background(), options)
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)

	return NewRelayService(n, 30, utils.Logger())
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
	require.True(t, reply)
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
	require.True(t, reply)

	err = d.DeleteV1Subscription(
		makeRequest(t),
		args,
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply)
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
	require.True(t, reply)

	// Wait for the subscription to be started
	time.Sleep(1 * time.Second)

	err = serviceA.PostV1Message(
		makeRequest(t),
		&RelayMessageArgs{
			Topic: "test",
			Message: RPCWakuRelayMessage{
				Payload: []byte("test"),
			},
		},
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply)

	// Wait for the message to be received
	time.Sleep(1 * time.Second)

	var messagesReply1 RelayMessagesReply
	err = serviceB.GetV1Messages(
		makeRequest(t),
		&TopicArgs{"test"},
		&messagesReply1,
	)
	require.NoError(t, err)
	require.Len(t, messagesReply1, 1)

	var messagesReply2 RelayMessagesReply
	err = serviceB.GetV1Messages(
		makeRequest(t),
		&TopicArgs{"test"},
		&messagesReply2,
	)
	require.NoError(t, err)
	require.Len(t, messagesReply2, 0)
}
