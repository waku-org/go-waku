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
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/stretchr/testify/require"
)

func makeWakuRelay(t *testing.T, topic relay.Topic, broadcaster v2.Broadcaster) (*relay.WakuRelay, *relay.Subscription, host.Host) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	relay, err := relay.NewWakuRelay(context.Background(), host, broadcaster)
	require.NoError(t, err)

	sub, err := relay.Subscribe(context.Background(), &topic)
	require.NoError(t, err)

	return relay, sub, host
}

func makeWakuFilter(t *testing.T, filters Filters) (*WakuFilter, host.Host) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	filterHandler := func(requestId string, msg pb.MessagePush) {
		for _, message := range msg.Messages {
			filters.Notify(message, requestId)
		}
	}

	filter := NewWakuFilter(context.Background(), host, false, filterHandler)

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
	var filters = make(Filters)
	var testTopic relay.Topic = "/waku/2/go/filter/test"
	testContentTopic := "TopicA"

	node1, host1 := makeWakuFilter(t, filters)
	defer node1.Stop()

	broadcaster := v2.NewBroadcaster(10)
	node2, sub2, host2 := makeWakuRelay(t, testTopic, broadcaster)
	defer node2.Stop()
	defer sub2.Unsubscribe()
	filterHandler := func(requestId string, msg pb.MessagePush) {
		for _, message := range msg.Messages {
			filters.Notify(message, requestId)
		}
	}

	node2Filter := NewWakuFilter(context.Background(), host2, true, filterHandler)
	broadcaster.Register(node2Filter.MsgC)

	host1.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err := host1.Peerstore().AddProtocols(host2.ID(), string(FilterID_v20beta1))
	require.NoError(t, err)

	contentFilter := &ContentFilter{
		Topic:         string(testTopic),
		ContentTopics: []string{testContentTopic},
	}
	sub, err := node1.Subscribe(context.Background(), *contentFilter, []FilterSubscribeOption{WithAutomaticPeerSelection()}...)
	require.NoError(t, err)

	// Sleep to make sure the filter is subscribed
	time.Sleep(2 * time.Second)

	ch := make(chan *protocol.Envelope, 1024)
	filters[sub.RequestID] = Filter{
		PeerID:         sub.Peer,
		Topic:          contentFilter.Topic,
		ContentFilters: contentFilter.ContentTopics,
		Chan:           ch,
	}
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		env := <-ch
		require.Equal(t, contentFilter.ContentTopics[0], env.Message().GetContentTopic())
	}()

	_, err = node2.Publish(context.Background(), &pb.WakuMessage{
		Payload:      []byte{1},
		Version:      0,
		ContentTopic: testContentTopic,
		Timestamp:    0,
	}, &testTopic)
	require.NoError(t, err)

	wg.Wait()

	wg.Add(1)
	go func() {
		select {
		case <-ch:
			require.Fail(t, "should not receive another message")
		case <-time.After(3 * time.Second):
			defer wg.Done()
		}
	}()

	_, err = node2.Publish(context.Background(), &pb.WakuMessage{
		Payload:      []byte{1},
		Version:      0,
		ContentTopic: "TopicB",
		Timestamp:    0,
	}, &testTopic)
	require.NoError(t, err)

	wg.Wait()
}
