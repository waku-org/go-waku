package rest

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/node"
	wakupeerstore "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func createNode(t *testing.T, opts ...node.WakuNodeOption) *node.WakuNode {
	node, err := node.New(opts...)
	require.NoError(t, err)

	err = node.Start(context.Background())
	require.NoError(t, err)

	return node
}

// node2 connects to node1
func twoFilterConnectedNodes(t *testing.T, pubSubTopic string) (*node.WakuNode, *node.WakuNode) {
	node1 := createNode(t, node.WithWakuFilterFullNode())  // full node filter
	node2 := createNode(t, node.WithWakuFilterLightNode()) // light node filter

	node2.Host().Peerstore().AddAddr(node1.Host().ID(), tests.GetHostAddress(node1.Host()), peerstore.PermanentAddrTTL)
	err := node2.Host().Peerstore().AddProtocols(node1.Host().ID(), filter.FilterSubscribeID_v20beta1)
	require.NoError(t, err)

	if pubSubTopic != "" {
		err = node2.Host().Peerstore().(*wakupeerstore.WakuPeerstoreImpl).SetPubSubTopics(node1.Host().ID(), []string{pubSubTopic})
		require.NoError(t, err)
	}

	return node1, node2
}

func getRequestId() string {
	return hex.EncodeToString(protocol.GenerateRequestID())
}

// test 400, 404 status code for ping rest endpoint
// both requests are not successful
func TestFilterPingFailure(t *testing.T) {
	node1, node2 := twoFilterConnectedNodes(t, "")
	defer func() {
		node1.Stop()
		node2.Stop()
	}()

	router := chi.NewRouter()
	_ = NewFilterService(node2, router, utils.Logger())

	// with malformed requestId
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/filter/v2/subscriptions/%s", "invalid_request_id"), nil)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestId:  []byte{},
		StatusDesc: "bad request id",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusBadRequest, rr.Code)

	// no subscription with peer
	var requestId filterRequestId = protocol.GenerateRequestID()
	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/filter/v2/subscriptions/%s", requestId), nil)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestId:  requestId,
		StatusDesc: "ping request failed",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

// create a filter subscription to the peer and try peer that peer
// both steps should be successful
func TestFilterSubscribeAndPing(t *testing.T) {
	pubsubTopic := "/waku/2/test/proto"
	contentTopics := []string{"test"}
	var requestId filterRequestId = protocol.GenerateRequestID()

	//
	node1, node2 := twoFilterConnectedNodes(t, pubsubTopic)
	defer func() {
		node1.Stop()
		node2.Stop()
	}()

	router := chi.NewRouter()
	_ = NewFilterService(node2, router, utils.Logger())

	// create subscription to peer
	rr := httptest.NewRecorder()
	reqReader := strings.NewReader(toString(t, filterSubscriptionRequest{
		RequestId:      requestId,
		PubsubTopic:    pubsubTopic,
		ContentFilters: contentTopics,
	}))
	req, _ := http.NewRequest(http.MethodPost, filterv2Subscribe, reqReader)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestId:  requestId,
		StatusDesc: "OK",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusOK, rr.Code)

	// trying pinging the peer once there is subscription to it
	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/filter/v2/subscriptions/%s", requestId), nil)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestId:  requestId,
		StatusDesc: "OK",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusOK, rr.Code)
}

// create subscription to peer
// delete the subscription to the peer with matching pubSub and contentTopic
func TestFilterSubscribeAndUnsubscribe(t *testing.T) {
	pubsubTopic := "/waku/2/test/proto"
	contentTopics := []string{"test"}
	var requestId filterRequestId = protocol.GenerateRequestID()

	//
	node1, node2 := twoFilterConnectedNodes(t, pubsubTopic)
	defer func() {
		node1.Stop()
		node2.Stop()
	}()

	router := chi.NewRouter()
	_ = NewFilterService(node2, router, utils.Logger())

	// create subscription to peer
	rr := httptest.NewRecorder()
	reqReader := strings.NewReader(toString(t, filterSubscriptionRequest{
		RequestId:      requestId,
		PubsubTopic:    pubsubTopic,
		ContentFilters: contentTopics,
	}))
	req, _ := http.NewRequest(http.MethodPost, filterv2Subscribe, reqReader)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestId:  requestId,
		StatusDesc: "OK",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusOK, rr.Code)

	// delete the subscription to the peer with matching pubSub and contentTopic
	requestId = protocol.GenerateRequestID()
	rr = httptest.NewRecorder()
	reqReader = strings.NewReader(toString(t, filterSubscriptionRequest{
		RequestId:      requestId,
		PubsubTopic:    pubsubTopic,
		ContentFilters: contentTopics,
	}))
	req, _ = http.NewRequest(http.MethodDelete, filterv2Subscribe, reqReader)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestId:  requestId,
		StatusDesc: "OK",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusOK, rr.Code)
}

// create 2 subscription from filter client to server
// make a unsubscribeAll request
// try pinging the peer, if 404 is returned then unsubscribeAll was successful
func TestFilterAllUnsubscribe(t *testing.T) {
	pubsubTopic := "/waku/2/test/proto"
	contentTopics1 := "ct_1"
	contentTopics2 := "ct_2"
	var requestId filterRequestId

	//
	node1, node2 := twoFilterConnectedNodes(t, pubsubTopic)
	defer func() {
		node1.Stop()
		node2.Stop()
	}()

	router := chi.NewRouter()
	_ = NewFilterService(node2, router, utils.Logger())

	// create 2 different subscription to peer
	for _, ct := range []string{contentTopics1, contentTopics2} {
		requestId = protocol.GenerateRequestID()
		rr := httptest.NewRecorder()
		reqReader := strings.NewReader(toString(t, filterSubscriptionRequest{
			RequestId:      requestId,
			PubsubTopic:    pubsubTopic,
			ContentFilters: []string{ct},
		}))
		req, _ := http.NewRequest(http.MethodPost, filterv2Subscribe, reqReader)
		router.ServeHTTP(rr, req)
		checkJSON(t, filterSubscriptionResponse{
			RequestId:  requestId,
			StatusDesc: "OK",
		}, getFilterResponse(t, rr.Body))
		require.Equal(t, http.StatusOK, rr.Code)
	}

	// delete all subscription to the peer
	requestId = protocol.GenerateRequestID()
	rr := httptest.NewRecorder()
	reqReader := strings.NewReader(toString(t, filterUnsubscribeAllRequest{
		RequestId: requestId,
	}))
	req, _ := http.NewRequest(http.MethodDelete, filterv2SubscribeAll, reqReader)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestId:  requestId,
		StatusDesc: "OK",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusOK, rr.Code)

	// check if all subscriptions are deleted to the peer are deleted
	requestId = protocol.GenerateRequestID()
	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/filter/v2/subscriptions/%s", requestId), nil)
	router.ServeHTTP(rr, req)
	checkJSON(t, filterSubscriptionResponse{
		RequestId:  requestId,
		StatusDesc: "ping request failed",
	}, getFilterResponse(t, rr.Body))
	require.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

func checkJSON(t *testing.T, expected, actual interface{}) {
	require.JSONEq(t, toString(t, expected), toString(t, actual))
}
func getFilterResponse(t *testing.T, body *bytes.Buffer) filterSubscriptionResponse {
	resp := filterSubscriptionResponse{}
	err := json.Unmarshal(body.Bytes(), &resp)
	require.NoError(t, err)
	return resp
}
func toString(t *testing.T, data interface{}) string {
	bytes, err := json.Marshal(data)
	require.NoError(t, err)
	return string(bytes)
}
