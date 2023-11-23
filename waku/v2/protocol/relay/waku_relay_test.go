package relay

import (
	"context"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

const defaultPubSubTopic = "/waku/2/go/relay/test"
const defaultContentTopic = "/test/10/my-app"

func TestWakuRelay(t *testing.T) {
	testTopic := defaultPubSubTopic

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	bcaster := NewBroadcaster(10)
	relay := NewWakuRelay(bcaster, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay.SetHost(host)
	err = relay.Start(context.Background())
	require.NoError(t, err)

	err = bcaster.Start(context.Background())
	require.NoError(t, err)
	defer relay.Stop()

	subs, err := relay.subscribe(context.Background(), protocol.NewContentFilter(testTopic))

	require.NoError(t, err)

	require.Equal(t, relay.IsSubscribed(testTopic), true)

	topics := relay.Topics()
	require.Equal(t, 1, len(topics))
	require.Equal(t, testTopic, topics[0])

	ctx, cancel := context.WithCancel(context.Background())
	bytesToSend := []byte{1}
	go func() {
		defer cancel()
		env := <-subs[0].Ch
		t.Log("received msg", logging.HexBytes("hash", env.Hash()))
	}()

	msg := &pb.WakuMessage{
		Payload:      bytesToSend,
		ContentTopic: "test",
	}
	_, err = relay.Publish(context.Background(), msg, WithPubSubTopic(testTopic))
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	err = relay.Unsubscribe(ctx, protocol.NewContentFilter(testTopic))
	require.NoError(t, err)
	<-ctx.Done()
}

func createRelayNode(t *testing.T) (host.Host, *WakuRelay) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)
	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	bcaster := NewBroadcaster(10)
	relay := NewWakuRelay(bcaster, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay.SetHost(host)
	err = bcaster.Start(context.Background())
	require.NoError(t, err)

	return host, relay
}

func TestGossipsubScore(t *testing.T) {
	testTopic := defaultPubSubTopic

	hosts := make([]host.Host, 5)
	relay := make([]*WakuRelay, 5)
	for i := 0; i < 5; i++ {
		hosts[i], relay[i] = createRelayNode(t)
		err := relay[i].Start(context.Background())
		require.NoError(t, err)
	}

	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if i == j {
				continue
			}

			hosts[i].Peerstore().AddAddrs(hosts[j].ID(), hosts[j].Addrs(), peerstore.PermanentAddrTTL)
			err := hosts[i].Connect(context.Background(), hosts[j].Peerstore().PeerInfo(hosts[j].ID()))
			require.NoError(t, err)
		}

		topicData, err := relay[i].subscribeToPubsubTopic(testTopic)
		require.NoError(t, err)
		go func() {
			for {
				_, err := topicData.subscription.Next(context.Background())
				if err != nil {
					t.Log(err)
				}
			}
		}()
	}

	time.Sleep(2 * time.Second)

	for i := 0; i < 5; i++ {
		require.Len(t, hosts[i].Network().Conns(), 4)
	}

	// We obtain the go-libp2p topic directly because we normally can't publish anything other than WakuMessages
	pubsubTopic, err := relay[0].upsertTopic(testTopic)
	require.NoError(t, err)

	// Removing validator from relayer0 to allow it to send invalid messages
	err = relay[0].pubsub.UnregisterTopicValidator(testTopic)
	require.NoError(t, err)

	for i := 0; i < 50; i++ {
		buf := make([]byte, 1000)
		_, err := rand.Read(buf)
		require.NoError(t, err)
		err = pubsubTopic.Publish(context.Background(), buf)
		require.NoError(t, err)
	}

	// long wait, must be higher than the configured decayInterval (how often score is updated)
	time.Sleep(20 * time.Second)

	// nodes[0] was blacklisted from all other peers, no connections
	require.Len(t, hosts[0].Network().Conns(), 0)

	for i := 1; i < 5; i++ {
		require.Len(t, hosts[i].Network().Conns(), 3)
	}
}

func TestMsgID(t *testing.T) {
	expectedMsgIDBytes := []byte{208, 214, 63, 55, 144, 6, 206, 39, 40, 251, 138, 74, 66, 168, 43, 32, 91, 94, 149, 122, 237, 198, 149, 87, 232, 156, 197, 34, 53, 131, 78, 112}

	topic := "abcde"
	msg := &pubsub_pb.Message{
		Data:  []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		Topic: &topic,
	}

	msgID := msgIDFn(msg)

	require.Equal(t, expectedMsgIDBytes, []byte(msgID))
}

func waitForTimeout(t *testing.T, ch chan *protocol.Envelope) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case env, ok := <-ch:
			if ok {
				t.Error("should not receive another message with payload", string(env.Message().Payload))
			}
		case <-time.After(2 * time.Second):
			// Timeout elapsed, all good
		}
	}()

	wg.Wait()
}

func waitForMsg(t *testing.T, ch chan *protocol.Envelope, cTopicExpected string) *sync.WaitGroup {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case env := <-ch:
			require.Equal(t, cTopicExpected, env.Message().GetContentTopic())
		case <-time.After(5 * time.Second):
			t.Error("Message timeout")
		}
	}()

	return &wg
}

func TestWakuRelayAutoShard(t *testing.T) {
	testcTopic := "/toychat/2/huilong/proto"
	testcTopic1 := "/toychat/1/huilong/proto"

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	bcaster := NewBroadcaster(10)
	relay := NewWakuRelay(bcaster, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay.SetHost(host)
	err = relay.Start(context.Background())
	require.NoError(t, err)

	err = bcaster.Start(context.Background())
	require.NoError(t, err)
	defer relay.Stop()
	defer bcaster.Stop()

	//Create a contentTopic level subscription
	subs, err := relay.subscribe(context.Background(), protocol.NewContentFilter("", testcTopic))
	require.NoError(t, err)
	require.Equal(t, relay.IsSubscribed(subs[0].contentFilter.PubsubTopic), true)

	sub, err := relay.GetSubscription(testcTopic)
	require.NoError(t, err)
	_, ok := sub.contentFilter.ContentTopics[testcTopic]
	require.Equal(t, true, ok)

	_, err = relay.GetSubscription(testcTopic1)
	require.Error(t, err)

	topics := relay.Topics()
	require.Equal(t, 1, len(topics))
	require.Equal(t, subs[0].contentFilter.PubsubTopic, topics[0])

	ctx, cancel := context.WithCancel(context.Background())
	bytesToSend := []byte{1}
	defer cancel()

	//Create a pubSub level subscription
	subs1, err := relay.subscribe(context.Background(), protocol.NewContentFilter(subs[0].contentFilter.PubsubTopic))
	require.NoError(t, err)

	msg := &pb.WakuMessage{
		Payload:      bytesToSend,
		ContentTopic: testcTopic,
	}
	_, err = relay.Publish(context.Background(), msg)
	require.NoError(t, err)

	wg := waitForMsg(t, subs[0].Ch, testcTopic)
	wg.Wait()

	wg = waitForMsg(t, subs1[0].Ch, testcTopic)
	wg.Wait()

	//Test publishing to different content-topic

	msg1 := &pb.WakuMessage{
		Payload:      bytesToSend,
		ContentTopic: testcTopic1,
	}

	_, err = relay.Publish(context.Background(), msg1, WithPubSubTopic(subs[0].contentFilter.PubsubTopic))
	require.NoError(t, err)

	wg = waitForMsg(t, subs1[0].Ch, testcTopic1)
	wg.Wait()

	//Should not receive message as subscription is for a different cTopic.
	waitForTimeout(t, subs[0].Ch)
	err = relay.Unsubscribe(ctx, protocol.NewContentFilter("", testcTopic))
	require.NoError(t, err)
	_, err = relay.GetSubscription(testcTopic)
	require.Error(t, err)
	_, err = relay.GetSubscription(testcTopic1)
	require.Error(t, err)

	topics = relay.Topics()
	require.Equal(t, 1, len(topics))
	require.Equal(t, subs[0].contentFilter.PubsubTopic, topics[0])
	wg2 := waitForMsg(t, subs1[0].Ch, testcTopic1)

	msg2 := &pb.WakuMessage{
		Payload:      bytesToSend,
		ContentTopic: testcTopic1,
		Timestamp:    utils.GetUnixEpoch(),
	}

	_, err = relay.Publish(context.Background(), msg2, WithPubSubTopic(subs[0].contentFilter.PubsubTopic))
	require.NoError(t, err)
	wg2.Wait()

	err = relay.Unsubscribe(ctx, protocol.NewContentFilter("", testcTopic))
	require.NoError(t, err)

	err = relay.Unsubscribe(ctx, protocol.NewContentFilter(subs[0].contentFilter.PubsubTopic))
	require.NoError(t, err)

}

func TestInvalidPushMessages(t *testing.T) {

	testTopic := defaultPubSubTopic
	testContentTopic := defaultContentTopic

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	bcaster := NewBroadcaster(10)
	relay := NewWakuRelay(bcaster, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay.SetHost(host)
	err = relay.Start(context.Background())
	require.NoError(t, err)

	err = bcaster.Start(context.Background())
	require.NoError(t, err)
	defer relay.Stop()

	ctx, ctxCancel := context.WithTimeout(context.Background(), 10*time.Second)

	subs, err := relay.subscribe(context.Background(), protocol.NewContentFilter(testTopic))
	require.NoError(t, err)

	// Test empty contentTopic
	_, err = relay.Publish(ctx, tests.CreateWakuMessage("", utils.GetUnixEpoch(), "test_payload"), WithPubSubTopic(testTopic))
	require.Error(t, err)

	// Test empty payload
	_, err = relay.Publish(ctx, tests.CreateWakuMessage(testContentTopic, utils.GetUnixEpoch(), ""), WithPubSubTopic(testTopic))
	require.Error(t, err)

	// Test empty contentTopic and payload
	_, err = relay.Publish(ctx, tests.CreateWakuMessage("", utils.GetUnixEpoch(), ""), WithPubSubTopic(testTopic))
	require.Error(t, err)

	// Test Meta size over limit
	message := tests.CreateWakuMessage(testContentTopic, utils.GetUnixEpoch(), "test_payload")
	message.Meta = make([]byte, 65)
	_, err = relay.Publish(ctx, message, WithPubSubTopic(testTopic))
	require.Error(t, err)

	err = relay.Unsubscribe(ctx, protocol.NewContentFilter(subs[0].contentFilter.PubsubTopic))
	require.NoError(t, err)

	ctxCancel()

}
