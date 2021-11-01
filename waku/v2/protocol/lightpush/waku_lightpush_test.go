package lightpush

import (
	"context"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/status-im/go-waku/tests"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/stretchr/testify/require"
)

func makeWakuRelay(t *testing.T, topic relay.Topic) (*relay.WakuRelay, *pubsub.Subscription, host.Host) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	relay, err := relay.NewWakuRelay(context.Background(), host)
	require.NoError(t, err)

	sub, _, err := relay.Subscribe(topic)
	require.NoError(t, err)

	return relay, sub, host
}

// Node1: Relay
// Node2: Relay+Lightpush
// Client that will lightpush a message
//
// Node1 and Node 2 are peers
// Client and Node 2 are peers
// Node 3 will use lightpush request, sending the message to Node2
//
// Client send a succesful message using lightpush
// Node2 receive the message and broadcast it
// Node1 receive the message
func TestWakuLightPush(t *testing.T) {
	var testTopic relay.Topic = "/waku/2/go/lightpush/test"
	node1, sub1, host1 := makeWakuRelay(t, testTopic)
	defer node1.Stop()
	defer sub1.Cancel()

	node2, sub2, host2 := makeWakuRelay(t, testTopic)
	defer node2.Stop()
	defer sub2.Cancel()

	ctx := context.Background()
	lightPushNode2 := NewWakuLightPush(ctx, host2, node2)
	err := lightPushNode2.Start()
	require.NoError(t, err)
	defer lightPushNode2.Stop()

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	clientHost, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	client := NewWakuLightPush(ctx, clientHost, nil)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), string(relay.WakuRelayID_v200))
	require.NoError(t, err)

	err = host2.Connect(ctx, host2.Peerstore().PeerInfo(host1.ID()))
	require.NoError(t, err)

	clientHost.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err = clientHost.Peerstore().AddProtocols(host2.ID(), string(LightPushID_v20beta1))
	require.NoError(t, err)

	msg1 := tests.CreateWakuMessage("test1", float64(0))
	msg2 := tests.CreateWakuMessage("test2", float64(1))

	req := new(pb.PushRequest)
	req.Message = msg1
	req.PubsubTopic = string(testTopic)

	// Wait for the mesh connection to happen between node1 and node2
	time.Sleep(2 * time.Second)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := sub1.Next(context.Background())
		require.NoError(t, err)

		_, err = sub1.Next(context.Background())
		require.NoError(t, err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := sub2.Next(context.Background())
		require.NoError(t, err)

		_, err = sub2.Next(context.Background())
		require.NoError(t, err)
	}()

	// Verifying succesful request
	resp, err := client.request(ctx, req)
	require.NoError(t, err)
	require.True(t, resp.IsSuccess)

	// Checking that msg hash is correct
	hash, err := client.Publish(ctx, msg2, &testTopic)
	require.NoError(t, err)
	require.Equal(t, protocol.NewEnvelope(msg2, string(testTopic)).Hash(), hash)
	wg.Wait()
}

func TestWakuLightPushStartWithoutRelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientHost, err := tests.MakeHost(context.Background(), 0, rand.Reader)
	require.NoError(t, err)

	client := NewWakuLightPush(ctx, clientHost, nil)

	err = client.Start()
	require.Errorf(t, err, "relay is required")
}

func TestWakuLightPushNoPeers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testTopic := relay.Topic("abc")

	clientHost, err := tests.MakeHost(context.Background(), 0, rand.Reader)
	require.NoError(t, err)
	client := NewWakuLightPush(ctx, clientHost, nil)

	_, err = client.Publish(ctx, tests.CreateWakuMessage("test", float64(0)), &testTopic)
	require.Errorf(t, err, "no suitable remote peers")
}
