package rpc

import (
	"context"
	"testing"

	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/stretchr/testify/require"
)

func TestWakuRpc(t *testing.T) {
	options := node.WithWakuStore(false, false)
	n, err := node.New(context.Background(), options)
	require.NoError(t, err)

	rpc := NewWakuRpc(n, "127.0.0.1", 8080)
	require.NotNil(t, rpc.server)
	require.Equal(t, rpc.server.Addr, "127.0.0.1:8080")
}
