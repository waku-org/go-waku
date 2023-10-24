package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/node"
	wakupeerstore "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func createLightPushNode(t *testing.T) *node.WakuNode {
	node, err := node.New(node.WithLightPush(), node.WithWakuRelay())
	require.NoError(t, err)

	err = node.Start(context.Background())
	require.NoError(t, err)

	return node
}

// node2 connects to node1
func twoLightPushConnectedNodes(t *testing.T, pubSubTopic string) (*node.WakuNode, *node.WakuNode) {
	node1 := createLightPushNode(t)
	node2 := createLightPushNode(t)

	node2.Host().Peerstore().AddAddr(node1.Host().ID(), tests.GetHostAddress(node1.Host()), peerstore.PermanentAddrTTL)
	err := node2.Host().Peerstore().AddProtocols(node1.Host().ID(), lightpush.LightPushID_v20beta1)
	require.NoError(t, err)
	err = node2.Host().Peerstore().(*wakupeerstore.WakuPeerstoreImpl).SetPubSubTopics(node1.Host().ID(), []string{pubSubTopic})
	require.NoError(t, err)

	return node1, node2
}

func TestLightpushMessagev1(t *testing.T) {
	pubSubTopic := "/waku/2/default-waku/proto"
	node1, node2 := twoLightPushConnectedNodes(t, pubSubTopic)
	defer func() {
		node1.Stop()
		node2.Stop()
	}()

	router := chi.NewRouter()
	serv := NewLightpushService(node2, router, utils.Logger())
	_ = serv

	msg := lightpushRequest{
		PubSubTopic: pubSubTopic,
		Message: &pb.WakuMessage{
			Payload:      []byte{1, 2, 3},
			ContentTopic: "abc",
			Version:      0,
			Timestamp:    utils.GetUnixEpoch(),
		},
	}
	msgJSONBytes, err := json.Marshal(msg)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, routeLightPushV1Messages, bytes.NewReader(msgJSONBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "true", rr.Body.String())
}
