package rpc

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/stretchr/testify/require"
)

func TestGetV1Info(t *testing.T) {
	var reply InfoReply

	request, err := http.NewRequest(http.MethodPost, "url", bytes.NewReader([]byte("")))
	require.NoError(t, err)

	wakuNode1, err := node.New(context.Background())
	require.NoError(t, err)
	defer wakuNode1.Stop()
	err = wakuNode1.Start()
	require.NoError(t, err)

	d := &DebugService{
		node: wakuNode1,
	}

	err = d.GetV1Info(request, &InfoArgs{}, &reply)
	require.NoError(t, err)
}
