package node

import (
	"context"
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/go-waku/tests"
	"github.com/stretchr/testify/require"
)

func TestWakuNode2(t *testing.T) {
	hostAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")

	key, err := tests.RandomHex(32)
	require.NoError(t, err)
	prvKey, err := crypto.HexToECDSA(key)
	require.NoError(t, err)

	ctx := context.Background()

	wakuNode, err := New(ctx,
		WithPrivateKey(prvKey),
		WithHostAddress([]*net.TCPAddr{hostAddr}),
		WithWakuRelay(),
	)
	require.NoError(t, err)

	err = wakuNode.Start()
	defer wakuNode.Stop()

	require.NoError(t, err)
}
