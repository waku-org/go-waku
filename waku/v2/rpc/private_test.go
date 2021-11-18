package rpc

import (
	"context"
	"testing"

	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/stretchr/testify/require"
)

func makePrivateService(t *testing.T) *PrivateService {
	n, err := node.New(context.Background(), node.WithWakuRelay())
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)

	return &PrivateService{n}
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

func TestGetV1AsymmetricKeypair(t *testing.T) {
	d := makePrivateService(t)
	defer d.node.Stop()

	var reply KeyPairReply
	err := d.GetV1AsymmetricKeypair(
		makeRequest(t),
		&Empty{},
		&reply,
	)
	require.NoError(t, err)
	require.NotEmpty(t, reply.PrivateKey)
	require.NotEmpty(t, reply.PulicKey)
}
