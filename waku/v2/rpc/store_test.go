package rpc

import (
	"context"
	"testing"

	logging "github.com/ipfs/go-log"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/stretchr/testify/require"
)

var log = logging.Logger("test")

func makeStoreService(t *testing.T) *StoreService {
	options := node.WithWakuStore(false, false)
	n, err := node.New(context.Background(), options)
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)
	return &StoreService{n, &log.SugaredLogger}
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
