package peermanager

import (
	"context"
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"strconv"
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

func makePeerManagerWithEventBus(t *testing.T, r *relay.WakuRelay, h host.Host) (*PeerManager, event.Bus) {
	// Host 1 used by peer manager
	pm := NewPeerManager(10, 20, utils.Logger())
	pm.SetHost(h)

	// Create a new relay event bus
	relayEvtBus := r.Events()

	// Subscribe to EventBus
	err := pm.SubscribeToRelayEvtBus(relayEvtBus)
	require.NoError(t, err)

	// Register necessary protocols
	pm.RegisterWakuProtocol(relay.WakuRelayID_v200, relay.WakuRelayENRField)
	return pm, relayEvtBus
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

	// Peermanager with event bus
	pm, _ := makePeerManagerWithEventBus(t, r, h1)
	pm.ctx = ctx
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

	// Check the original topic was removed from Peer Manager
	_, ok = pm.subRelayTopics[pubSubTopic]
	require.False(t, ok)

}

func TestHandlePeerTopicEvent(t *testing.T) {
	log := utils.Logger()
	pubSubTopic := "/waku/2/go/pm/test"
	ctx := context.Background()

	// Relay and Host1
	r, h1, _ := makeWakuRelay(t, log)
	err := r.Start(ctx)
	require.NoError(t, err)

	// Relay and Host2
	r2, h2, _ := makeWakuRelay(t, log)
	err = r2.Start(ctx)
	require.NoError(t, err)

	// Peermanager with event bus
	pm, eventBus := makePeerManagerWithEventBus(t, r, h1)
	pm.ctx = ctx
	go pm.connectivityLoop(ctx)

	// Add h2 peer
	_, err = pm.AddPeer(getAddr(h2), wps.Static, []string{""}, relay.WakuRelayID_v200)
	require.NoError(t, err)

	h1.Peerstore().AddAddr(h2.ID(), tests.GetHostAddress(h2), peerstore.PermanentAddrTTL)
	err = h1.Peerstore().AddProtocols(h2.ID(), relay.WakuRelayID_v200)
	require.NoError(t, err)

	// Subscribe to Pubsub topic which also emits relay.PEER_JOINED
	_, err = r.Subscribe(ctx, protocol.NewContentFilter(pubSubTopic))
	require.NoError(t, err)

	// peerEvt to find: relay.PEER_JOINED, relay.PEER_LEFT
	peerEvt := relay.EvtPeerTopic{
		PubsubTopic: pubSubTopic,
		PeerID:      h1.ID(),
		State:       relay.PEER_JOINED,
	}

	emitter, err := eventBus.Emitter(new(relay.EvtPeerTopic))
	require.NoError(t, err)

	err = emitter.Emit(peerEvt)
	require.NoError(t, err)

	// Call the appropriate handler
	for i := 0; i < 2; i++ {
		select {
		case e := <-pm.sub.Out():
			switch e := e.(type) {
			case relay.EvtPeerTopic:
				{
					log.Info("Handling topic event...")
					peerEvt := (relay.EvtPeerTopic)(e)
					pm.handlerPeerTopicEvent(peerEvt)
					for id := range pm.host.Peerstore().(*wps.WakuPeerstoreImpl).PeersByPubSubTopic(pubSubTopic) {
						log.Info("hosts before", zap.String("id", strconv.Itoa(id)))
					}
				}
			case relay.EvtRelaySubscribed:
				{
					log.Info("Handling subscribe event...")
					eventDetails := (relay.EvtRelaySubscribed)(e)
					pm.handleNewRelayTopicSubscription(eventDetails.Topic, eventDetails.TopicInst)
				}
			default:
				require.Fail(t, "unexpected event arrived")
			}

		case <-ctx.Done():
			require.Fail(t, "closed channel")
		}
	}

	// Evaluate topic health - unhealthy at first, because no peers connected
	peerTopic := pm.subRelayTopics[peerEvt.PubsubTopic]
	//pm.checkAndUpdateTopicHealth(topic)
	require.Equal(t, TopicHealth(UnHealthy), peerTopic.healthStatus)

	// peerEvt to find: relay.PEER_JOINED, relay.PEER_LEFT
	peerEvt2 := relay.EvtPeerTopic{
		PubsubTopic: pubSubTopic,
		PeerID:      h2.ID(),
		State:       relay.PEER_JOINED,
	}

	err = emitter.Emit(peerEvt2)
	require.NoError(t, err)

	// Call the appropriate handler
	select {
	case e := <-pm.sub.Out():
		switch e := e.(type) {
		case relay.EvtPeerTopic:
			{
				log.Info("Handling topic event...")
				peerEvt := (relay.EvtPeerTopic)(e)
				pm.handlerPeerTopicEvent(peerEvt)
				for id := range pm.host.Peerstore().(*wps.WakuPeerstoreImpl).PeersByPubSubTopic(pubSubTopic) {
					log.Info("hosts after", zap.String("id", strconv.Itoa(id)))
				}
			}
		default:
			require.Fail(t, "unexpected event arrived")
		}

	case <-ctx.Done():
		require.Fail(t, "closed channel")
	}

	// Evaluate topic health - unhealthy at first, because no peers connected
	peerTopic = pm.subRelayTopics[peerEvt.PubsubTopic]

	for id := range peerTopic.topic.ListPeers() {
		log.Info("peers", zap.String("ID", strconv.Itoa(id)))
	}

	pm.checkAndUpdateTopicHealth(pm.subRelayTopics[peerEvt.PubsubTopic])
	require.Equal(t, TopicHealth(MinimallyHealthy), peerTopic.healthStatus)

}
