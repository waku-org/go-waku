package filter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/waku-org/go-waku/waku/v2/protocol/filter/pb"

	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

func (s *FilterTestSuite) TestWakuFilter() {
	// Initial subscribe
	s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Should be received
	s.waitForMsg(&WakuMsg{s.testTopic, s.testContentTopic, "first"})

	// Wrong content topic
	s.waitForTimeout(&WakuMsg{s.testTopic, "TopicB", "second"})

	_, err := s.lightNode.Unsubscribe(s.ctx, s.contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	// Should not receive after unsubscribe
	s.waitForTimeout(&WakuMsg{s.testTopic, s.testContentTopic, "third"})

	// Two new subscriptions with same [peer, contentFilter]
	s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	secondSub := s.getSub(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Assert that we have 2 subscriptions now
	s.Require().Equal(len(s.lightNode.Subscriptions()), 2)

	// Should be received on both subscriptions
	s.waitForMsg(&WakuMsg{s.testTopic, s.testContentTopic, "fourth"})

	s.waitForMsgFromChan(&WakuMsg{s.testTopic, s.testContentTopic, "fifth"}, secondSub[0].C)

	s.waitForMsg(nil)
	s.waitForMsgFromChan(nil, secondSub[0].C)

	// Unsubscribe from second sub only
	_, err = s.lightNode.UnsubscribeWithSubscription(s.ctx, secondSub[0])
	s.Require().NoError(err)

	// Should still receive
	s.waitForMsg(&WakuMsg{s.testTopic, s.testContentTopic, "sixth"})

	// Unsubscribe from first sub only
	_, err = s.lightNode.UnsubscribeWithSubscription(s.ctx, s.subDetails[0])
	s.Require().NoError(err)

	s.Require().Equal(len(s.lightNode.Subscriptions()), 0)

	// Should not receive after unsubscribe
	s.waitForTimeout(&WakuMsg{s.testTopic, s.testContentTopic, "seventh"})
}

func (s *FilterTestSuite) TestPubSubSingleContentTopic() {
	// Initial subscribe
	s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Message should be received
	s.waitForMsg(&WakuMsg{s.testTopic, s.testContentTopic, "test_msg"})

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestPubSubMultiContentTopic() {

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 10 seconds

	messages := prepareData(3, false, true, false, nil)

	// Subscribe
	for _, m := range messages {
		s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestMultiPubSubMultiContentTopic() {

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 20 seconds

	s.MakeWakuFilterLightNode()
	s.StartLightNode()

	s.MakeWakuFilterFullNode(s.testTopic, true)

	// Connect nodes
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)
	err := s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	messages := prepareData(2, true, true, false, nil)

	// Subscribe
	for _, m := range messages {
		s.subDetails = append(s.subDetails, s.getSub(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())...)
		s.Log.Info("Subscribing ", zap.String("PubSubTopic", m.pubSubTopic))
		_, err := s.relayNode.Subscribe(context.Background(), protocol.NewContentFilter(m.pubSubTopic))
		s.Require().NoError(err)
	}

	// Debug to see subscriptions in light node
	for _, sub := range s.subDetails {
		s.Log.Info("Light Node subscription ", zap.String("PubSubTopic", sub.ContentFilter.PubsubTopic), zap.String("ContentTopic", sub.ContentFilter.ContentTopicsList()[0]))
	}

	// All messages should be received
	s.waitForMessages(messages)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestPubSubMultiOverlapContentTopic() {

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 20 seconds

	messages := prepareData(10, false, true, true, nil)

	// Subscribe
	for _, m := range messages {
		s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestSubscriptionRefresh() {

	messages := prepareData(2, false, false, true, nil)

	// Initial subscribe
	s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Repeat the same subscribe
	s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Both messages should be received
	s.waitForMessages(messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestContentTopicsLimit() {
	var maxContentTopics = pb.MaxContentTopicsPerRequest

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 10 seconds

	// Detect existing content topics from previous test
	if len(s.contentFilter.PubsubTopic) > 0 {
		existingTopics := len(s.contentFilter.ContentTopicsList())
		if existingTopics > 0 {
			maxContentTopics = maxContentTopics - existingTopics
		}
	}

	messages := prepareData(maxContentTopics+1, false, true, true, nil)

	// Subscribe
	for _, m := range messages[:len(messages)-1] {
		s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// All messages within limit should get received
	s.waitForMessages(messages[:len(messages)-1])

	// Adding over the limit contentTopic should fail
	for _, sub := range s.subDetails {
		if sub.ContentFilter.PubsubTopic == messages[len(messages)-1].pubSubTopic {
			sub.Add(messages[len(messages)-1].contentTopic)
			_, err := s.lightNode.Subscribe(s.ctx, sub.ContentFilter, WithPeer(s.fullNodeHost.ID()))
			s.Require().Error(err)
		}
	}

	// Unsubscribe for cleanup
	for _, m := range messages {
		_ = s.unsubscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestSubscribeErrorHandling() {
	var messages []WakuMsg

	// Prepare data
	messages = append(messages, WakuMsg{
		pubSubTopic:  "",
		contentTopic: s.testContentTopic,
		payload:      "N/A",
	})

	messages = append(messages, WakuMsg{
		pubSubTopic:  s.testTopic,
		contentTopic: "",
		payload:      "N/A",
	})

	// Subscribe with empty pubsub
	s.contentFilter = protocol.ContentFilter{PubsubTopic: messages[0].pubSubTopic, ContentTopics: protocol.NewContentTopicSet(messages[0].contentTopic)}
	_, err := s.lightNode.Subscribe(s.ctx, s.contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().Error(err)

	// Subscribe with empty content topic
	s.contentFilter = protocol.ContentFilter{PubsubTopic: messages[1].pubSubTopic, ContentTopics: protocol.NewContentTopicSet(messages[1].contentTopic)}
	_, err = s.lightNode.Subscribe(s.ctx, s.contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().Error(err)

}

func (s *FilterTestSuite) TestMultipleFullNodeSubscriptions() {
	log := utils.Logger()
	s.Log = log
	s.wg = &sync.WaitGroup{}

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds

	fullNodeIDHex := make([]byte, hex.EncodedLen(len([]byte(s.fullNodeHost.ID()))))
	_ = hex.Encode(fullNodeIDHex, []byte(s.fullNodeHost.ID()))

	s.Log.Info("Already subscribed to", zap.String("fullNode", string(fullNodeIDHex)))

	// This will overwrite values with the second node info
	s.MakeWakuFilterFullNode(s.testTopic, false)

	// Connect to second full and relay node
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)
	err := s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	fullNodeIDHex = make([]byte, hex.EncodedLen(len([]byte(s.fullNodeHost.ID()))))
	_ = hex.Encode(fullNodeIDHex, []byte(s.fullNodeHost.ID()))

	s.Log.Info("Subscribing to second", zap.String("fullNode", string(fullNodeIDHex)))

	// Subscribe to the second full node
	s.contentFilter = protocol.ContentFilter{PubsubTopic: s.testTopic, ContentTopics: protocol.NewContentTopicSet(s.testContentTopic)}
	_, err = s.lightNode.Subscribe(s.ctx, s.contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)
}

func (s *FilterTestSuite) TestSubscribeMultipleLightNodes() {

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds

	lightNodeData := s.GetWakuFilterLightNode()
	lightNode2 := lightNodeData.lightNode
	err := lightNode2.Start(context.Background())
	s.Require().NoError(err)

	// Connect node2
	lightNode2.h.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)

	messages := prepareData(2, true, true, true, nil)

	// Subscribe separately: light node 1 -> full node
	contentFilter := protocol.ContentFilter{PubsubTopic: messages[0].pubSubTopic, ContentTopics: protocol.NewContentTopicSet(messages[0].contentTopic)}
	_, err = s.lightNode.Subscribe(s.ctx, contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	// Subscribe separately: light node 2 -> full node
	contentFilter2 := protocol.ContentFilter{PubsubTopic: messages[1].pubSubTopic, ContentTopics: protocol.NewContentTopicSet(messages[1].contentTopic)}
	_, err = lightNode2.Subscribe(s.ctx, contentFilter2, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	// Unsubscribe
	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

	_, err = lightNode2.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestSubscribeFullNode2FullNode() {

	var (
		testTopic        = "/waku/2/go/filter/test2"
		testContentTopic = "TopicB"
	)

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 10*time.Second)

	nodeData := s.GetWakuFilterFullNode(testTopic, false)
	fullNode2 := nodeData.fullNode

	// Connect nodes
	fullNode2.h.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)

	// Get stream
	stream, err := fullNode2.h.NewStream(s.ctx, s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	// Prepare subscribe request
	subscribeRequest := &pb.FilterSubscribeRequest{
		FilterSubscribeType: pb.FilterSubscribeRequest_SUBSCRIBE,
		PubsubTopic:         &testTopic,
		ContentTopics:       []string{testContentTopic},
	}

	// Subscribe full node 2 -> full node 1
	fullNode2.subscribe(s.ctx, stream, subscribeRequest)

	// Check the pubsub topic related to the first node is stored within the second node
	pubsubTopics, hasTopics := fullNode2.subscriptions.Get(s.fullNodeHost.ID())
	s.Require().True(hasTopics)

	// Check the pubsub topic is what we have set
	contentTopics, hasTestPubsubTopic := pubsubTopics[testTopic]
	s.Require().True(hasTestPubsubTopic)

	// Check the content topic is what we have set
	_, hasTestContentTopic := contentTopics[testContentTopic]
	s.Require().True(hasTestContentTopic)

}

func (s *FilterTestSuite) TestIsSubscriptionAlive() {
	messages := prepareData(2, false, true, false, nil)

	// Subscribe with the first message only
	s.subscribe(messages[0].pubSubTopic, messages[0].contentTopic, s.fullNodeHost.ID())

	// IsSubscriptionAlive returns no error for the first message
	err := s.lightNode.IsSubscriptionAlive(s.ctx, s.subDetails[0])
	s.Require().NoError(err)

	// Create new host/peer - not related to any node
	host, err := tests.MakeHost(context.Background(), 54321, rand.Reader)
	s.Require().NoError(err)

	// Alter the existing peer ID in sub details
	s.subDetails[0].PeerID = host.ID()

	// IsSubscriptionAlive returns error for the second message, peer ID doesn't match
	err = s.lightNode.IsSubscriptionAlive(s.ctx, s.subDetails[0])
	s.Require().Error(err)

}

func (s *FilterTestSuite) TestFilterSubscription() {
	contentFilter := protocol.ContentFilter{PubsubTopic: s.testTopic, ContentTopics: protocol.NewContentTopicSet(s.testContentTopic)}

	// Subscribe
	s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Returns no error and SubscriptionDetails for existing subscription
	_, err := s.lightNode.FilterSubscription(s.fullNodeHost.ID(), contentFilter)
	s.Require().NoError(err)

	otherFilter := protocol.ContentFilter{PubsubTopic: "34583495", ContentTopics: protocol.NewContentTopicSet("sjfa402")}

	// Returns error and nil SubscriptionDetails for non existent subscription
	nonSubscription, err := s.lightNode.FilterSubscription(s.fullNodeHost.ID(), otherFilter)
	s.Require().Error(err)
	s.Require().Nil(nonSubscription)

	// Create new host/peer - not related to any node
	host, err := tests.MakeHost(context.Background(), 54321, rand.Reader)
	s.Require().NoError(err)

	// Returns error and nil SubscriptionDetails for unrelated host/peer
	nonSubscription, err = s.lightNode.FilterSubscription(host.ID(), contentFilter)
	s.Require().Error(err)
	s.Require().Nil(nonSubscription)

}

func (s *FilterTestSuite) TestHandleFilterSubscribeOptions() {
	contentFilter := protocol.ContentFilter{PubsubTopic: s.testTopic, ContentTopics: protocol.NewContentTopicSet(s.testContentTopic)}

	// Subscribe
	s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// With valid peer
	opts := []FilterSubscribeOption{WithPeer(s.fullNodeHost.ID())}

	// Positive case
	_, _, err := s.lightNode.handleFilterSubscribeOptions(s.ctx, contentFilter, opts)
	s.Require().NoError(err)

	addr := s.fullNodeHost.Addrs()[0]

	// Combine mutually exclusive options
	opts = []FilterSubscribeOption{WithPeer(s.fullNodeHost.ID()), WithPeerAddr(addr)}

	// Should fail on wrong option combination
	_, _, err = s.lightNode.handleFilterSubscribeOptions(s.ctx, contentFilter, opts)
	s.Require().Error(err)

}
