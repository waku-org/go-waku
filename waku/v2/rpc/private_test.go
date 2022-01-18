package rpc

import (
	"context"
	"testing"

	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/stretchr/testify/require"
)

func makePrivateService(t *testing.T) *PrivateService {
	n, err := node.New(context.Background(), node.WithWakuRelay())
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)

	return &PrivateService{n, &log.SugaredLogger}
}

func TestGetV1SymmetricKey(t *testing.T) {
	d := makePrivateService(t)
	defer d.node.Stop()

	var reply SymmetricKeyReply
	err := d.GetV1SymmetricKey(
		makeRequest(t),
		&Empty{},
		&reply,
	)
	require.NoError(t, err)
	require.NotEmpty(t, reply.Key)
}

func TestGetV1AsymmetricKey(t *testing.T) {
	d := makePrivateService(t)
	defer d.node.Stop()

	var reply KeyPairReply
	err := d.GetV1AsymmetricKeypair(
		makeRequest(t),
		&Empty{},
		&reply,
	)
	require.NoError(t, err)
	require.NotEmpty(t, reply.PulicKey)
	require.NotEmpty(t, reply.PrivateKey)
}

func TestPostV1SymmetricMessage(t *testing.T) {
	d := makePrivateService(t)
	defer d.node.Stop()

	var reply SuccessReply
	err := d.PostV1SymmetricMessage(
		makeRequest(t),
		&SymmetricMessageArgs{
			Topic:   "test",
			Message: pb.WakuMessage{Payload: []byte("test")},
			SymKey:  "abc",
		},
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply.Success)
}

func TestPostV1AsymmetricMessage(t *testing.T) {
	d := makePrivateService(t)
	defer d.node.Stop()

	var reply SuccessReply
	err := d.PostV1AsymmetricMessage(
		makeRequest(t),
		&AsymmetricMessageArgs{
			Topic:     "test",
			Message:   pb.WakuMessage{Payload: []byte("test")},
			PublicKey: "045ded6a56c88173e87a88c55b96956964b1bd3351b5fcb70950a4902fbc1bc0ceabb0ac846c3a4b8f2f6024c0e19f0a7f6a4865035187de5463f34012304fc7c5",
		},
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply.Success)
}
