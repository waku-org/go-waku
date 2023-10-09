package rest

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func createNode(t *testing.T, opts ...node.WakuNodeOption) *node.WakuNode {
	node, err := node.New()
	require.NoError(t, err)

	err = node.Start(context.Background())
	require.NoError(t, err)

	return node
}

// node2 connects to node1
func twoFilterConnectedNodes(t *testing.T) (*node.WakuNode, *node.WakuNode) {
	node1 := createNode(t, node.WithWakuFilterFullNode())  // full node filter
	node2 := createNode(t, node.WithWakuFilterLightNode()) // light node filter

	node2.Host().Peerstore().AddAddr(node1.Host().ID(), tests.GetHostAddress(node1.Host()), peerstore.PermanentAddrTTL)
	err := node2.Host().Peerstore().AddProtocols(node1.Host().ID(), filter.FilterSubscribeID_v20beta1)
	require.NoError(t, err)

	return node1, node2
}

func getRequestId() string {
	return hex.EncodeToString(protocol.GenerateRequestID())
}

// test 400, 404, 200 status code for ping rest endpoint
func TestFilterPing(t *testing.T) {
	node1, node2 := twoFilterConnectedNodes(t)
	defer func() {
		node1.Stop()
		node2.Stop()
	}()

	router := chi.NewRouter()
	_ = NewFilterService(node2, router, utils.Logger())

	requestId := "invalid_request_id"

	// with malformed requestId
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/filter/v2/subscriptions/%s", requestId), nil)
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "true", rr.Body.String())

	// without peer
	requestId = getRequestId()
	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/filter/v2/subscriptions/%s", requestId), nil)
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusServiceUnavailable, rr.Code)
	require.Equal(t, http.StatusServiceUnavailable, rr.Code)

	// with peer
	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/filter/v2/subscriptions/%s", requestId), nil)
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}
