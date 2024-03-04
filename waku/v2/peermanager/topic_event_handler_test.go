package peermanager

import (
	"context"
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
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
	"time"
)

func makeWakuRelay(t *testing.T, log *zap.Logger) (*relay.WakuRelay, host.Host, relay.Broadcaster) {

	broadcaster := relay.NewBroadcaster(10)
	require.NoError(t, broadcaster.Start(context.Background()))

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	h, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	broadcaster.RegisterForAll()

	r := relay.NewWakuRelay(broadcaster, 0, timesource.NewDefaultClock(),
		prometheus.DefaultRegisterer, log)

	r.SetHost(h)

	return r, h, broadcaster
}

func makePeerManagerWithEventBus(t *testing.T, r *relay.WakuRelay, h *host.Host) (*PeerManager, event.Bus) {
	// Host 1 used by peer manager
	pm := NewPeerManager(10, 20, utils.Logger())
	pm.SetHost(*h)

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
	pm, _ := makePeerManagerWithEventBus(t, r, &h1)
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

	hosts := make([]host.Host, 5)
	relays := make([]*relay.WakuRelay, 5)

	for i := 0; i < 5; i++ {
		relays[i], hosts[i], _ = makeWakuRelay(t, log)
		err := relays[i].Start(ctx)
		require.NoError(t, err)
	}

	// Create peer manager instance with the first hosts
	pm, eventBus := makePeerManagerWithEventBus(t, relays[0], &hosts[0])
	pm.ctx = ctx
	pm.RegisterWakuProtocol(relay.WakuRelayID_v200, relay.WakuRelayENRField)

	// Connect host[0] with all other hosts to reach 4 connections
	for i := 1; i < 5; i++ {
		pm.host.Peerstore().AddAddrs(hosts[i].ID(), hosts[i].Addrs(), peerstore.PermanentAddrTTL)
		err := pm.host.Connect(ctx, hosts[i].Peerstore().PeerInfo(hosts[i].ID()))
		require.NoError(t, err)
		err = pm.host.Peerstore().(wps.WakuPeerstore).SetDirection(hosts[i].ID(), network.DirOutbound)
		require.NoError(t, err)

	}

	go pm.connectivityLoop(ctx)

	time.Sleep(2 * time.Second)

	if len(pm.host.Peerstore().(*wps.WakuPeerstoreImpl).PeersByPubSubTopic(pubSubTopic)) == 0 {
		log.Info("No peers for the topic yet")
	}

	// Subscribe to Pubsub topic
	_, err := relays[0].Subscribe(ctx, protocol.NewContentFilter(pubSubTopic))
	require.NoError(t, err)

	// peerEvt to find: relay.PEER_JOINED, relay.PEER_LEFT
	peerEvt := relay.EvtPeerTopic{
		PubsubTopic: pubSubTopic,
		PeerID:      hosts[1].ID(),
		State:       relay.PEER_JOINED,
	}

	emitter, err := eventBus.Emitter(new(relay.EvtPeerTopic))
	require.NoError(t, err)

	err = emitter.Emit(peerEvt)
	require.NoError(t, err)

	// Process subscribe event and first PEER_JOINED event
	for i := 0; i < 2; i++ {
		select {
		case e := <-pm.sub.Out():
			switch e := e.(type) {
			case relay.EvtPeerTopic:
				{
					log.Info("Handling topic event...")
					peerEvt := (relay.EvtPeerTopic)(e)
					pm.handlerPeerTopicEvent(peerEvt)
					for _, peer := range pm.host.Peerstore().(*wps.WakuPeerstoreImpl).PeersByPubSubTopic(pubSubTopic) {
						log.Info("hosts before", zap.String("peer", peer.String()))
						log.Info("peer", zap.String("connectedness", string(rune(pm.host.Network().Connectedness(peer)))))
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
	peerTopic, ok := pm.subRelayTopics[pubSubTopic]
	if ok {
		log.Info("New topic subscribed", zap.String("topic", peerTopic.topic.String()))
	}
	pm.checkAndUpdateTopicHealth(peerTopic)
	require.Equal(t, TopicHealth(UnHealthy), peerTopic.healthStatus)
	time.Sleep(2 * time.Second)

	// Process second to fourth PEER_JOINED event
	for i := 2; i < 4; i++ {

		peerEvt2 := relay.EvtPeerTopic{
			PubsubTopic: pubSubTopic,
			PeerID:      hosts[i].ID(),
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
					for _, peer := range pm.host.Peerstore().(*wps.WakuPeerstoreImpl).PeersByPubSubTopic(pubSubTopic) {
						log.Info("hosts after", zap.String("id", peer.String()))
						log.Info("peer", zap.String("connectedness", string(rune(pm.host.Network().Connectedness(peer)))))
					}
				}
			default:
				require.Fail(t, "unexpected event arrived")
			}

		case <-ctx.Done():
			require.Fail(t, "closed channel")
		}

		// Evaluate topic health - unhealthy at first, because D > #peers connected
		pm.checkAndUpdateTopicHealth(peerTopic)
		require.Equal(t, TopicHealth(UnHealthy), peerTopic.healthStatus)
		time.Sleep(2 * time.Second)
	}

	// Evaluate topic health - should reach minimal health - 4 peers connected
	for id := range peerTopic.topic.ListPeers() {
		log.Info("peers joined", zap.String("ID", strconv.Itoa(id)))
	}
	pm.checkAndUpdateTopicHealth(peerTopic)

	peersIn, peersOut := pm.getRelayPeers()
	log.Info("IDS peers", zap.String("in ", strconv.Itoa(len(peersIn))), zap.String("out", strconv.Itoa(len(peersOut))))

	notConnectedPeers := pm.getNotConnectedPers(pubSubTopic)
	log.Info("IDS peers", zap.String("not connected", strconv.Itoa(len(notConnectedPeers))))

	require.Equal(t, TopicHealth(MinimallyHealthy), peerTopic.healthStatus)
}
