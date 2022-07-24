package rest

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/stretchr/testify/require"
)

func TestGetV1Info(t *testing.T) {
	wakuNode1, err := node.New(context.Background())
	require.NoError(t, err)
	defer wakuNode1.Stop()
	err = wakuNode1.Start()
	require.NoError(t, err)

	d := &DebugService{
		node: wakuNode1,
	}

	request, err := http.NewRequest(http.MethodPost, ROUTE_DEBUG_INFOV1, bytes.NewReader([]byte("")))
	require.NoError(t, err)

	rr := httptest.NewRecorder()

	d.getV1Info(rr, request)

	require.Equal(t, http.StatusOK, rr.Code)
}
