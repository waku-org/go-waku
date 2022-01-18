package rpc

import (
	"context"
	"testing"

	"github.com/status-im/go-waku/tests"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/stretchr/testify/require"
)

func makeStoreService(t *testing.T) *StoreService {
	options := node.WithWakuStore(false, false)
	n, err := node.New(context.Background(), options)
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)
	return &StoreService{n, tests.Logger()}
}

func TestStoreGetV1Messages(t *testing.T) {
	var reply StoreMessagesReply

	s := makeStoreService(t)

	err := s.GetV1Messages(
		makeRequest(t),
		&StoreMessagesArgs{},
		&reply,
	)
	require.NoError(t, err)
	require.NotEmpty(t, reply.Error)
}
