package lightpush

import (
	"context"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func makeWakuRelay(t *testing.T, pusubTopic string) (*relay.WakuRelay, *relay.Subscription, host.Host) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	b := relay.NewBroadcaster(10)
	require.NoError(t, b.Start(context.Background()))
	relay := relay.NewWakuRelay(b, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay.SetHost(host)
	require.NoError(t, err)
	err = relay.Start(context.Background())
	require.NoError(t, err)

	sub, err := relay.SubscribeToTopic(context.Background(), pusubTopic)
	require.NoError(t, err)

	return relay, sub, host
}

// Node1: Relay
// Node2: Relay+Lightpush
// Client that will lightpush a message
//
// Node1 and Node 2 are peers
// Client and Node 2 are peers
// Client will use lightpush request, sending the message to Node2
//
// Client send a successful message using lightpush
// Node2 receive the message and broadcast it
// Node1 receive the message
func TestWakuLightPush(t *testing.T) {
	testTopic := "/waku/2/go/lightpush/test"
	node1, sub1, host1 := makeWakuRelay(t, testTopic)
	defer node1.Stop()
	defer sub1.Unsubscribe()

	node2, sub2, host2 := makeWakuRelay(t, testTopic)
	defer node2.Stop()
	defer sub2.Unsubscribe()

	ctx := context.Background()
	lightPushNode2 := NewWakuLightPush(node2, nil, prometheus.DefaultRegisterer, utils.Logger())
	lightPushNode2.SetHost(host2)
	err := lightPushNode2.Start(ctx)
	require.NoError(t, err)
	defer lightPushNode2.Stop()

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	clientHost, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	client := NewWakuLightPush(nil, nil, prometheus.DefaultRegisterer, utils.Logger())
	client.SetHost(clientHost)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), relay.WakuRelayID_v200)
	require.NoError(t, err)

	err = host2.Connect(ctx, host2.Peerstore().PeerInfo(host1.ID()))
	require.NoError(t, err)

	clientHost.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err = clientHost.Peerstore().AddProtocols(host2.ID(), LightPushID_v20beta1)
	require.NoError(t, err)

	msg1 := tests.CreateWakuMessage("test1", utils.GetUnixEpoch())
	msg2 := tests.CreateWakuMessage("test2", utils.GetUnixEpoch())

	req := new(pb.PushRequest)
	req.Message = msg1
	req.PubsubTopic = string(testTopic)

	// Wait for the mesh connection to happen between node1 and node2
	time.Sleep(2 * time.Second)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sub1.Ch
		<-sub1.Ch
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sub2.Ch
		<-sub2.Ch
	}()

	// Verifying successful request
	resp, err := client.request(ctx, req)
	require.NoError(t, err)
	require.True(t, resp.IsSuccess)

	// Checking that msg hash is correct
	hash, err := client.PublishToTopic(ctx, msg2, testTopic)
	require.NoError(t, err)
	require.Equal(t, protocol.NewEnvelope(msg2, utils.GetUnixEpoch(), string(testTopic)).Hash(), hash)
	wg.Wait()
}

func TestWakuLightPushStartWithoutRelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientHost, err := tests.MakeHost(context.Background(), 0, rand.Reader)
	require.NoError(t, err)
	client := NewWakuLightPush(nil, nil, prometheus.DefaultRegisterer, utils.Logger())
	client.SetHost(clientHost)
	err = client.Start(ctx)

	require.Errorf(t, err, "relay is required")
}

func TestWakuLightPushNoPeers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testTopic := "abc"

	clientHost, err := tests.MakeHost(context.Background(), 0, rand.Reader)
	require.NoError(t, err)
	client := NewWakuLightPush(nil, nil, prometheus.DefaultRegisterer, utils.Logger())
	client.SetHost(clientHost)
	_, err = client.PublishToTopic(ctx, tests.CreateWakuMessage("test", utils.GetUnixEpoch()), testTopic)
	require.Errorf(t, err, "no suitable remote peers")
}

// Node1: Relay
// Node2: Relay+Lightpush
// Client that will lightpush a message
//
// Node1 and Node 2 are peers
// Client and Node 2 are peers
// Client will use lightpush request, sending the message to Node2
//
// Client send a successful message using lightpush
// Node2 receive the message and broadcast it
// Node1 receive the message
func TestWakuLightPushAutoSharding(t *testing.T) {
	contentTopic := "0/test/1/testTopic/proto"
	cTopic1, err := protocol.StringToContentTopic(contentTopic)
	require.NoError(t, err)
	//Computing pubSubTopic only for filterFullNode.
	pubSubTopicInst := protocol.GetShardFromContentTopic(cTopic1, protocol.GenerationZeroShardsCount)
	pubSubTopic := pubSubTopicInst.String()
	node1, sub1, host1 := makeWakuRelay(t, pubSubTopic)
	defer node1.Stop()
	defer sub1.Unsubscribe()

	node2, sub2, host2 := makeWakuRelay(t, pubSubTopic)
	defer node2.Stop()
	defer sub2.Unsubscribe()

	ctx := context.Background()
	lightPushNode2 := NewWakuLightPush(node2, nil, prometheus.DefaultRegisterer, utils.Logger())
	lightPushNode2.SetHost(host2)
	err = lightPushNode2.Start(ctx)
	require.NoError(t, err)
	defer lightPushNode2.Stop()

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	clientHost, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	client := NewWakuLightPush(nil, nil, prometheus.DefaultRegisterer, utils.Logger())
	client.SetHost(clientHost)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), relay.WakuRelayID_v200)
	require.NoError(t, err)

	err = host2.Connect(ctx, host2.Peerstore().PeerInfo(host1.ID()))
	require.NoError(t, err)

	clientHost.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err = clientHost.Peerstore().AddProtocols(host2.ID(), LightPushID_v20beta1)
	require.NoError(t, err)

	msg1 := tests.CreateWakuMessage(contentTopic, utils.GetUnixEpoch())
	msg2 := tests.CreateWakuMessage(contentTopic, utils.GetUnixEpoch())

	// Wait for the mesh connection to happen between node1 and node2
	time.Sleep(2 * time.Second)
	var wg sync.WaitGroup
	t.Log("host2 peers ", host2.Network().Peers())
	t.Log("host1 peers ", host1.Network().Peers())

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sub1.Ch
		<-sub1.Ch
		t.Logf("Received msgs on relay1")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sub2.Ch
		<-sub2.Ch
		t.Logf("Received msgs on relay2")
	}()

	// Verifying successful request
	hash1, err := client.Publish(ctx, msg1)
	require.NoError(t, err)
	require.Equal(t, protocol.NewEnvelope(msg1, utils.GetUnixEpoch(), string(pubSubTopic)).Hash(), hash1)

	// Checking that msg hash is correct
	hash, err := client.PublishToTopic(ctx, msg2, "")
	require.NoError(t, err)
	require.Equal(t, protocol.NewEnvelope(msg2, utils.GetUnixEpoch(), string(pubSubTopic)).Hash(), hash)
	wg.Wait()

}
