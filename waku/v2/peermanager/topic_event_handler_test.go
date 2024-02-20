package peermanager

import (
	"context"
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"testing"
)

func makeWakuRelay(t *testing.T, log *zap.Logger) (*relay.WakuRelay, host.Host, relay.Broadcaster) {

	broadcaster := relay.NewBroadcaster(10)
	require.NoError(t, broadcaster.Start(context.Background()))

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	relay := relay.NewWakuRelay(broadcaster, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, log)
	relay.SetHost(host)

	return relay, host, broadcaster
}

func TestSubscribeToRelayEvtBus(t *testing.T) {
	log := utils.Logger()

	// Host 1
	r, h1, _ := makeWakuRelay(t, log)

	// Host 1 used by peer manager
	pm := NewPeerManager(10, 20, utils.Logger())
	pm.SetHost(h1)

	// Create a new relay event bus
	relayEvtBus := r.Events()

	// Subscribe to EventBus
	err := pm.SubscribeToRelayEvtBus(relayEvtBus)
	require.NoError(t, err)

}

func TestHandleRelayTopicSubscription(t *testing.T) {
	log := utils.Logger()
	pubSubTopic := "/waku/2/go/pm/test"
	ctx := context.Background()

	// Relay and Host
	r, h1, _ := makeWakuRelay(t, log)
	err := r.Start(ctx)
	require.NoError(t, err)

	// Host 1 used by peer manager
	pm := NewPeerManager(10, 20, utils.Logger())
	pm.SetHost(h1)

	// Create a new relay event bus
	relayEvtBus := r.Events()

	// Subscribe to EventBus
	err = pm.SubscribeToRelayEvtBus(relayEvtBus)
	require.NoError(t, err)

	// Register necessary protocols and connect
	pm.ctx = ctx
	pm.RegisterWakuProtocol(relay.WakuRelayID_v200, relay.WakuRelayENRField)
	go pm.connectivityLoop(ctx)

	// Subscribe to Pubsub topic
	_, err = r.Subscribe(ctx, protocol.NewContentFilter(pubSubTopic))
	require.NoError(t, err)

	// Call the appropriate handler
	select {
	case e := <-pm.sub.Out():
		switch e := e.(type) {
		case relay.EvtRelaySubscribed:
			{
				eventDetails := (relay.EvtRelaySubscribed)(e)
				pm.handleNewRelayTopicSubscription(eventDetails.Topic, eventDetails.TopicInst)
			}
		default:
			require.Fail(t, "unexpected event arrived")
		}

	case <-ctx.Done():
		require.Fail(t, "closed channel")
	}

	// Check Peer Manager knows about the topic
	_, ok := pm.subRelayTopics[pubSubTopic]
	require.True(t, ok)

	// UnSubscribe from Pubsub topic
	err = r.Unsubscribe(ctx, protocol.NewContentFilter(pubSubTopic))
	require.NoError(t, err)

	// Call the appropriate handler
	select {
	case e := <-pm.sub.Out():
		switch e := e.(type) {
		case relay.EvtRelayUnsubscribed:
			{
				eventDetails := (relay.EvtRelayUnsubscribed)(e)
				pm.handleNewRelayTopicUnSubscription(eventDetails.Topic)
			}
		default:
			require.Fail(t, "unexpected event arrived")
		}

	case <-ctx.Done():
		require.Fail(t, "closed channel")
	}

	// Check Peer Manager knows about the topic
	_, ok = pm.subRelayTopics[pubSubTopic]
	require.False(t, ok)

}
