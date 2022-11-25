package rpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestWakuRpc(t *testing.T) {
	options := node.WithWakuStore(false, nil)
	n, err := node.New(context.Background(), options)
	require.NoError(t, err)

	rpc := NewWakuRpc(n, "127.0.0.1", 8080, true, true, 30, utils.Logger())
	require.NotNil(t, rpc.server)
	require.Equal(t, rpc.server.Addr, "127.0.0.1:8080")
}
