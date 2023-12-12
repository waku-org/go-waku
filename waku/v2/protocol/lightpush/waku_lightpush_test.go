package lightpush

import (
	"context"
	"crypto/rand"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
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

	sub, err := relay.Subscribe(context.Background(), protocol.NewContentFilter(pusubTopic))
	require.NoError(t, err)

	return relay, sub[0], host
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

	msg2 := tests.CreateWakuMessage("test2", utils.GetUnixEpoch())

	// Wait for the mesh connection to happen between node1 and node2
	time.Sleep(2 * time.Second)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sub1.Ch
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sub2.Ch
	}()

	var lpOptions []Option
	lpOptions = append(lpOptions, WithPubSubTopic(testTopic))
	lpOptions = append(lpOptions, WithPeer(host2.ID()))

	// Checking that msg hash is correct
	hash, err := client.Publish(ctx, msg2, lpOptions...)
	require.NoError(t, err)
	require.Equal(t, protocol.NewEnvelope(msg2, *utils.GetUnixEpoch(), string(testTopic)).Hash(), hash)
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
	var lpOptions []Option
	lpOptions = append(lpOptions, WithPubSubTopic(testTopic))

	_, err = client.Publish(ctx, tests.CreateWakuMessage("test", utils.GetUnixEpoch()), lpOptions...)
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

	// Wait for the mesh connection to happen between node1 and node2
	time.Sleep(2 * time.Second)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sub1.Ch
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sub2.Ch

	}()
	var lpOptions []Option
	lpOptions = append(lpOptions, WithPeer(host2.ID()))
	// Verifying successful request
	hash1, err := client.Publish(ctx, msg1, lpOptions...)
	require.NoError(t, err)
	require.Equal(t, protocol.NewEnvelope(msg1, *utils.GetUnixEpoch(), string(pubSubTopic)).Hash(), hash1)

	wg.Wait()

}

func TestWakuLightPushCornerCases(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testTopic := "/waku/2/go/lightpush/test"
	testContentTopic := "/test/10/my-lp-app/proto"

	// Prepare peer manager instance to include in test
	pm := peermanager.NewPeerManager(10, 10, utils.Logger())

	node1, sub1, host1 := makeWakuRelay(t, testTopic)
	defer node1.Stop()
	defer sub1.Unsubscribe()

	node2, sub2, host2 := makeWakuRelay(t, testTopic)
	defer node2.Stop()
	defer sub2.Unsubscribe()

	lightPushNode2 := NewWakuLightPush(node2, pm, prometheus.DefaultRegisterer, utils.Logger())
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

	msg2 := tests.CreateWakuMessage(testContentTopic, utils.GetUnixEpoch())

	// Wait for the mesh connection to happen between node1 and node2
	time.Sleep(2 * time.Second)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sub1.Ch
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sub2.Ch
	}()

	var lpOptions []Option
	lpOptions = append(lpOptions, WithPubSubTopic(testTopic))
	lpOptions = append(lpOptions, WithPeer(host2.ID()))

	// Check that msg publish has passed for nominal case
	_, err = client.Publish(ctx, msg2, lpOptions...)
	require.NoError(t, err)

	// Test error case with nil message
	_, err = client.Publish(ctx, nil, lpOptions...)
	require.Error(t, err)

	// Create new "dummy" host - not related to any node
	host3, err := tests.MakeHost(context.Background(), 12345, rand.Reader)
	require.NoError(t, err)

	var lpOptions2 []Option

	// Test error case with empty options
	_, err = client.Publish(ctx, msg2, lpOptions2...)
	require.Error(t, err)

	// Test error case with unrelated host
	_, err = client.Publish(ctx, msg2, WithPubSubTopic(testTopic), WithPeer(host3.ID()))
	require.Error(t, err)

	// Test corner case with default pubSub topic
	_, err = client.Publish(ctx, msg2, WithDefaultPubsubTopic(), WithPeer(host2.ID()))
	require.NoError(t, err)

	wg.Wait()

	// Test situation when cancel func is nil
	lightPushNode2.cancel = nil
}
