package relay

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestWakuRelay(t *testing.T) {
	testTopic := "/waku/2/go/relay/test"

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

	_, err = relay.subscribe(testTopic)
	sub := bcaster.Register(testTopic)
	defer sub.Unsubscribe()
	require.NoError(t, err)

	require.Equal(t, relay.IsSubscribed(testTopic), true)

	topics := relay.Topics()
	require.Equal(t, 1, len(topics))
	require.Equal(t, testTopic, topics[0])

	ctx, cancel := context.WithCancel(context.Background())
	bytesToSend := []byte{1}
	go func() {
		defer cancel()

		<-sub.Ch

	}()

	msg := &pb.WakuMessage{
		Payload:      bytesToSend,
		Version:      0,
		ContentTopic: "test",
		Timestamp:    0,
	}
	_, err = relay.PublishToTopic(context.Background(), msg, testTopic)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	err = relay.Unsubscribe(ctx, testTopic)
	require.NoError(t, err)

	<-ctx.Done()
}

func createRelayNode(t *testing.T) (host.Host, *WakuRelay) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)
	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	relay := NewWakuRelay(nil, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay.SetHost(host)
	return host, relay
}

func TestGossipsubScore(t *testing.T) {
	testTopic := "/waku/2/go/relay/test"

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

		sub, err := relay[i].subscribe(testTopic)
		require.NoError(t, err)
		go func() {
			for {
				_, err := sub.Next(context.Background())
				if err != nil {
					fmt.Println(err)
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
