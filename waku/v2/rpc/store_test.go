package rpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func makeStoreService(t *testing.T) *StoreService {
	options := node.WithWakuStore(false, false)
	n, err := node.New(context.Background(), options)
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)
	return &StoreService{n, utils.Logger()}
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
