package peermanager

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"testing"
)

func TestSubscribeToRelayEvtBus(t *testing.T) {
	log := utils.Logger()

	// Host 1
	r, h1, _ := tests.MakeWakuRelay(t, log)

	// Host 1 used by peer manager
	pm := NewPeerManager(10, 20, utils.Logger())
	pm.SetHost(h1)

	// Create a new relay event bus
	relayEvtBus := r.Events()

	// Subscribe to EventBus
	err := pm.SubscribeToRelayEvtBus(relayEvtBus)
	require.NoError(t, err)

}

func TestHandleNewRelayTopicSubscription(t *testing.T) {
	log := utils.Logger()
	pubSubTopic := "/waku/2/go/pm/test"
	ctx := context.Background()

	// Relay and Host
	r, h1, _ := tests.MakeWakuRelay(t, log)
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
}
