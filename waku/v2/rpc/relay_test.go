package rpc

import (
	"context"
	"testing"

	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/stretchr/testify/require"
)

func makeRelayService(t *testing.T) *RelayService {
	options := node.WithWakuRelay()
	n, err := node.New(context.Background(), options)
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)
	return &RelayService{n}
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
