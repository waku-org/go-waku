package filter

import (
	"context"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/status-im/go-waku/tests"
	v2 "github.com/status-im/go-waku/waku/v2"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/stretchr/testify/require"
)

func makeWakuRelay(t *testing.T, topic string, broadcaster v2.Broadcaster) (*relay.WakuRelay, *relay.Subscription, host.Host) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	relay, err := relay.NewWakuRelay(context.Background(), host, broadcaster, 0, tests.Logger())
	require.NoError(t, err)

	sub, err := relay.SubscribeToTopic(context.Background(), topic)
	require.NoError(t, err)

	return relay, sub, host
}

func makeWakuFilter(t *testing.T) (*WakuFilter, host.Host) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	filter, _ := NewWakuFilter(context.Background(), host, false, tests.Logger())

	return filter, host
}

// Node1: Filter subscribed to content topic A
// Node2: Relay + Filter
//
// Node1 and Node2 are peers
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

	node1, host1 := makeWakuFilter(t)
	defer node1.Stop()

	broadcaster := v2.NewBroadcaster(10)
	node2, sub2, host2 := makeWakuRelay(t, testTopic, broadcaster)
	defer node2.Stop()
	defer sub2.Unsubscribe()

	node2Filter, _ := NewWakuFilter(ctx, host2, true, tests.Logger())
	broadcaster.Register(node2Filter.MsgC)

	host1.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err := host1.Peerstore().AddProtocols(host2.ID(), string(FilterID_v20beta1))
	require.NoError(t, err)

	contentFilter := &ContentFilter{
		Topic:         string(testTopic),
		ContentTopics: []string{testContentTopic},
	}

	_, f, err := node1.Subscribe(ctx, *contentFilter, WithPeer(node2Filter.h.ID()))
	require.NoError(t, err)

	// Sleep to make sure the filter is subscribed
	time.Sleep(2 * time.Second)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		env := <-f.Chan
		require.Equal(t, contentFilter.ContentTopics[0], env.Message().GetContentTopic())
	}()

	_, err = node2.PublishToTopic(ctx, tests.CreateWakuMessage(testContentTopic, 0), testTopic)
	require.NoError(t, err)

	wg.Wait()

	wg.Add(1)
	go func() {
		select {
		case <-f.Chan:
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
		case <-f.Chan:
			require.Fail(t, "should not receive another message")
		case <-time.After(1 * time.Second):
			defer wg.Done()
		case <-ctx.Done():
			require.Fail(t, "test exceeded allocated time")
		}
	}()

	err = node1.Unsubscribe(ctx, *contentFilter, node2Filter.h.ID())
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

	node1, host1 := makeWakuFilter(t)

	broadcaster := v2.NewBroadcaster(10)
	node2, sub2, host2 := makeWakuRelay(t, testTopic, broadcaster)
	defer node2.Stop()
	defer sub2.Unsubscribe()

	node2Filter, _ := NewWakuFilter(ctx, host2, true, tests.Logger(), WithTimeout(3*time.Second))
	broadcaster.Register(node2Filter.MsgC)

	host1.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err := host1.Peerstore().AddProtocols(host2.ID(), string(FilterID_v20beta1))
	require.NoError(t, err)

	contentFilter := &ContentFilter{
		Topic:         string(testTopic),
		ContentTopics: []string{testContentTopic},
	}

	_, f, err := node1.Subscribe(ctx, *contentFilter, WithPeer(node2Filter.h.ID()))
	require.NoError(t, err)

	// Simulate there's been a failure before
	node2Filter.subscribers.FlagAsFailure(host1.ID())

	// Sleep to make sure the filter is subscribed
	time.Sleep(2 * time.Second)

	_, ok := node2Filter.subscribers.failedPeers[host1.ID()]
	require.True(t, ok)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		env := <-f.Chan
		require.Equal(t, contentFilter.ContentTopics[0], env.Message().GetContentTopic())

		// Failure is removed
		_, ok := node2Filter.subscribers.failedPeers[host1.ID()]
		require.False(t, ok)

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
	_, ok = node2Filter.subscribers.failedPeers[host1.ID()]
	require.True(t, ok)

	time.Sleep(3 * time.Second)

	_, err = node2.PublishToTopic(ctx, tests.CreateWakuMessage(testContentTopic, 2), testTopic)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)
	_, ok = node2Filter.subscribers.failedPeers[host1.ID()]
	require.False(t, ok) // Failed peer has been removed

	for subscriber := range node2Filter.subscribers.Items() {
		if subscriber.peer == node1.h.ID() {
			require.Fail(t, "Subscriber should not exist")
		}
	}

}
