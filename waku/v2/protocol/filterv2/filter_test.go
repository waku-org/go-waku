package filterv2

import (
	"context"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func makeWakuRelay(t *testing.T, topic string, broadcaster v2.Broadcaster) (*relay.WakuRelay, *relay.Subscription, host.Host) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	relay := relay.NewWakuRelay(host, broadcaster, 0, timesource.NewDefaultClock(), utils.Logger())
	err = relay.Start(context.Background())
	require.NoError(t, err)

	sub, err := relay.SubscribeToTopic(context.Background(), topic)
	require.NoError(t, err)

	return relay, sub, host
}

func makeWakuFilterLightNode(t *testing.T) (*WakuFilterLightnode, host.Host) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	filterPush := NewWakuFilterLightnode(host, v2.NewBroadcaster(10), timesource.NewDefaultClock(), utils.Logger())
	err = filterPush.Start(context.Background())
	require.NoError(t, err)

	return filterPush, host
}

// Node1: Filter subscribed to content topic A
// Node2: Relay + Filter
//
// # Node1 and Node2 are peers
//
// Node2 send a successful message with topic A
// Node1 receive the message
//
// Node2 send a successful message with topic B
// Node1 doesn't receive the message
func TestWakuFilter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds
	defer cancel()

	testTopic := "/waku/2/go/filter/test"
	testContentTopic := "TopicA"

	node1, host1 := makeWakuFilterLightNode(t)
	defer node1.Stop()

	broadcaster := v2.NewBroadcaster(10)
	node2, sub2, host2 := makeWakuRelay(t, testTopic, broadcaster)
	defer node2.Stop()
	defer sub2.Unsubscribe()

	node2Filter := NewWakuFilterFullnode(host2, broadcaster, timesource.NewDefaultClock(), utils.Logger())
	err := node2Filter.Start(ctx)
	require.NoError(t, err)

	broadcaster.Register(&testTopic, node2Filter.MessageChannel())

	host1.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err = host1.Peerstore().AddProtocols(host2.ID(), FilterSubscribeID_v20beta1)
	require.NoError(t, err)

	contentFilter := ContentFilter{
		Topic:         string(testTopic),
		ContentTopics: []string{testContentTopic},
	}

	subscriptionChannel, err := node1.Subscribe(ctx, contentFilter, WithPeer(node2Filter.h.ID()))
	require.NoError(t, err)

	// Sleep to make sure the filter is subscribed
	time.Sleep(2 * time.Second)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		env := <-subscriptionChannel.C
		require.Equal(t, contentFilter.ContentTopics[0], env.Message().GetContentTopic())
	}()

	_, err = node2.PublishToTopic(ctx, tests.CreateWakuMessage(testContentTopic, 0), testTopic)
	require.NoError(t, err)

	wg.Wait()

	wg.Add(1)
	go func() {
		select {
		case <-subscriptionChannel.C:
			require.Fail(t, "should not receive another message")
		case <-time.After(1 * time.Second):
			defer wg.Done()
		case <-ctx.Done():
			require.Fail(t, "test exceeded allocated time")
		}
	}()

	_, err = node2.PublishToTopic(ctx, tests.CreateWakuMessage("TopicB", 1), testTopic)
	require.NoError(t, err)

	wg.Wait()

	wg.Add(1)
	go func() {
		select {
		case <-subscriptionChannel.C:
			require.Fail(t, "should not receive another message")
		case <-time.After(1 * time.Second):
			defer wg.Done()
		case <-ctx.Done():
			require.Fail(t, "test exceeded allocated time")
		}
	}()

	_, err = node1.Unsubscribe(ctx, contentFilter, Peer(node2Filter.h.ID()))
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	_, err = node2.PublishToTopic(ctx, tests.CreateWakuMessage(testContentTopic, 2), testTopic)
	require.NoError(t, err)
	wg.Wait()
}

func TestWakuFilterPeerFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds
	defer cancel()

	testTopic := "/waku/2/go/filter/test"
	testContentTopic := "TopicA"

	node1, host1 := makeWakuFilterLightNode(t)

	broadcaster := v2.NewBroadcaster(10)
	node2, sub2, host2 := makeWakuRelay(t, testTopic, broadcaster)
	defer node2.Stop()
	defer sub2.Unsubscribe()

	node2Filter := NewWakuFilterFullnode(host2, v2.NewBroadcaster(10), timesource.NewDefaultClock(), utils.Logger(), filter.WithTimeout(5*time.Second))
	err := node2Filter.Start(ctx)
	require.NoError(t, err)

	broadcaster.Register(&testTopic, node2Filter.MessageChannel())

	host1.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err = host1.Peerstore().AddProtocols(host2.ID(), FilterPushID_v20beta1)
	require.NoError(t, err)

	contentFilter := &ContentFilter{
		Topic:         string(testTopic),
		ContentTopics: []string{testContentTopic},
	}

	f, err := node1.Subscribe(ctx, *contentFilter, WithPeer(node2Filter.h.ID()))
	require.NoError(t, err)

	// Simulate there's been a failure before
	node2Filter.subscriptions.FlagAsFailure(host1.ID())

	// Sleep to make sure the filter is subscribed
	time.Sleep(2 * time.Second)

	require.True(t, node2Filter.subscriptions.IsFailedPeer(host1.ID()))

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		env := <-f.C
		require.Equal(t, contentFilter.ContentTopics[0], env.Message().GetContentTopic())

		// Failure is removed
		require.False(t, node2Filter.subscriptions.IsFailedPeer(host1.ID()))

	}()

	_, err = node2.PublishToTopic(ctx, tests.CreateWakuMessage(testContentTopic, 0), testTopic)
	require.NoError(t, err)

	wg.Wait()

	// Kill the subscriber
	host1.Close()

	time.Sleep(1 * time.Second)

	_, err = node2.PublishToTopic(ctx, tests.CreateWakuMessage(testContentTopic, 1), testTopic)
	require.NoError(t, err)

	// TODO: find out how to eliminate this sleep
	time.Sleep(1 * time.Second)
	require.True(t, node2Filter.subscriptions.IsFailedPeer(host1.ID()))

	time.Sleep(2 * time.Second)

	_, err = node2.PublishToTopic(ctx, tests.CreateWakuMessage(testContentTopic, 2), testTopic)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	require.True(t, node2Filter.subscriptions.IsFailedPeer(host1.ID())) // Failed peer has been removed
	require.False(t, node2Filter.subscriptions.Has(host1.ID()))         // Failed peer has been removed
}
