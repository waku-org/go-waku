package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func makeRelayService(t *testing.T, mux *chi.Mux) *RelayService {
	options := node.WithWakuRelayAndMinPeers(0)
	n, err := node.New(options)
	require.NoError(t, err)
	err = n.Start(context.Background())
	require.NoError(t, err)

	return NewRelayService(n, mux, 3, utils.Logger())
}

func TestPostV1Message(t *testing.T) {
	router := chi.NewRouter()

	_ = makeRelayService(t, router)

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: "abc",
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}
	msgJsonBytes, err := json.Marshal(msg)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/relay/v1/messages/test", bytes.NewReader(msgJsonBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "true", rr.Body.String())
}

func TestRelaySubscription(t *testing.T) {
	router := chi.NewRouter()

	d := makeRelayService(t, router)

	go d.Start(context.Background())
	defer d.Stop()

	// Wait for node to start
	time.Sleep(500 * time.Millisecond)

	topics := []string{"test"}
	topicsJSONBytes, err := json.Marshal(topics)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/relay/v1/subscriptions", bytes.NewReader(topicsJSONBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "true", rr.Body.String())

	// Test max messages in subscription
	now := utils.GetUnixEpoch()
	d.runner.broadcaster.Submit(protocol.NewEnvelope(tests.CreateWakuMessage("test", now+1), now, "test"))
	d.runner.broadcaster.Submit(protocol.NewEnvelope(tests.CreateWakuMessage("test", now+2), now, "test"))
	d.runner.broadcaster.Submit(protocol.NewEnvelope(tests.CreateWakuMessage("test", now+3), now, "test"))

	// Wait for the messages to be processed
	time.Sleep(500 * time.Millisecond)

	require.Len(t, d.messages["test"], 3)

	d.runner.broadcaster.Submit(protocol.NewEnvelope(tests.CreateWakuMessage("test", now+4), now+4, "test"))

	time.Sleep(500 * time.Millisecond)

	// Should only have 3 messages
	require.Len(t, d.messages["test"], 3)

	// Test deletion
	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, "/relay/v1/subscriptions", bytes.NewReader(topicsJSONBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "true", rr.Body.String())
	require.Len(t, d.messages["test"], 0)

}

func TestRelayGetV1Messages(t *testing.T) {
	router := chi.NewRouter()

	serviceA := makeRelayService(t, router)
	go serviceA.Start(context.Background())
	defer serviceA.Stop()
	serviceB := makeRelayService(t, router)
	go serviceB.Start(context.Background())
	defer serviceB.Stop()

	hostInfo, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", serviceB.node.Host().ID().Pretty()))
	require.NoError(t, err)

	var addr multiaddr.Multiaddr
	for _, a := range serviceB.node.Host().Addrs() {
		addr = a.Encapsulate(hostInfo)
		break
	}
	err = serviceA.node.DialPeerWithMultiAddress(context.Background(), addr)
	require.NoError(t, err)

	// Wait for the dial to complete
	time.Sleep(1 * time.Second)

	topics := []string{"test"}
	topicsJSONBytes, err := json.Marshal(topics)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/relay/v1/subscriptions", bytes.NewReader(topicsJSONBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	// Wait for the subscription to be started
	time.Sleep(1 * time.Second)

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: "test",
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}
	msgJsonBytes, err := json.Marshal(msg)
	require.NoError(t, err)

	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/relay/v1/messages/test", bytes.NewReader(msgJsonBytes))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	// Wait for the message to be received
	time.Sleep(1 * time.Second)

	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/relay/v1/messages/test", bytes.NewReader([]byte{}))
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var messages []*pb.WakuMessage
	err = json.Unmarshal(rr.Body.Bytes(), &messages)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	rr = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/relay/v1/messages/test", bytes.NewReader([]byte{}))
	router.ServeHTTP(rr, req)

	err = json.Unmarshal(rr.Body.Bytes(), &messages)
	require.NoError(t, err)
	require.Len(t, messages, 0)
}
