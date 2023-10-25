package filter

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

func (s *FilterTestSuite) TestWakuFilter() {
	// Initial subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Should be received
	s.waitForMsg(func() {
		s.publishMsg(s.testTopic, s.testContentTopic, "first")
	}, s.subDetails[0].C)

	// Wrong content topic
	s.waitForTimeout(func() {
		s.publishMsg(s.testTopic, "TopicB", "second")
	}, s.subDetails[0].C)

	_, err := s.lightNode.Unsubscribe(s.ctx, s.contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	// Should not receive after unsubscribe
	s.waitForTimeout(func() {
		s.publishMsg(s.testTopic, s.testContentTopic, "third")
	}, s.subDetails[0].C)

	// Two new subscriptions with same [peer, contentFilter]
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())
	secondSub := s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Assert that we have 2 subscriptions now
	s.Require().Equal(len(s.lightNode.Subscriptions()), 2)

	// Should be received on both subscriptions
	s.waitForMsg(func() {
		s.publishMsg(s.testTopic, s.testContentTopic, "fourth")
	}, s.subDetails[0].C)

	s.waitForMsg(func() {
		s.publishMsg(s.testTopic, s.testContentTopic, "fifth")
	}, secondSub[0].C)

	s.waitForMsg(nil, s.subDetails[0].C)
	s.waitForMsg(nil, secondSub[0].C)

	// Unsubscribe from second sub only
	_, err = s.lightNode.UnsubscribeWithSubscription(s.ctx, secondSub[0])
	s.Require().NoError(err)

	// Should still receive
	s.waitForMsg(func() {
		s.publishMsg(s.testTopic, s.testContentTopic, "sixth")
	}, s.subDetails[0].C)

	// Unsubscribe from first sub only
	_, err = s.lightNode.UnsubscribeWithSubscription(s.ctx, s.subDetails[0])
	s.Require().NoError(err)

	s.Require().Equal(len(s.lightNode.Subscriptions()), 0)

	// Should not receive after unsubscribe
	s.waitForTimeout(func() {
		s.publishMsg(s.testTopic, s.testContentTopic, "seventh")
	}, s.subDetails[0].C)
}

func (s *FilterTestSuite) TestPubSubSingleContentTopic() {
	// Initial subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Message should be received
	s.waitForMsg(func() {
		s.publishMsg(s.testTopic, s.testContentTopic, "test_msg")
	}, s.subDetails[0].C)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestPubSubMultiContentTopic() {

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 10 seconds

	messages := prepareData(3, false, true, false)

	// Subscribe
	for _, m := range messages {
		s.subDetails = s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestMultiPubSubMultiContentTopic() {

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 20 seconds

	s.lightNode = s.makeWakuFilterLightNode(true, true)

	s.relayNode, s.fullNode = s.makeWakuFilterFullNode(s.testTopic, true)

	// Connect nodes
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)
	err := s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	messages := prepareData(2, true, true, false)

	// Subscribe
	for _, m := range messages {
		s.subDetails = append(s.subDetails, s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())...)
		s.log.Info("Subscribing ", zap.String("PubSubTopic", m.pubSubTopic))
		_, err := s.relayNode.Subscribe(context.Background(), protocol.NewContentFilter(m.pubSubTopic))
		s.Require().NoError(err)
	}

	// Debug to see subscriptions in light node
	for _, sub := range s.subDetails {
		s.log.Info("Light Node subscription ", zap.String("PubSubTopic", sub.ContentFilter.PubsubTopic), zap.String("ContentTopic", sub.ContentFilter.ContentTopicsList()[0]))
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestPubSubMultiOverlapContentTopic() {

	// Create test context
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 20 seconds
	s.ctx = ctx
	s.ctxCancel = cancel

	messages := prepareData(10, false, true, true)

	// Subscribe
	for _, m := range messages {
		s.subDetails = s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestSubscriptionRefresh() {

	messages := prepareData(2, false, false, true)

	// Initial subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Repeat the same subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Both messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestContentTopicsLimit() {
	var maxContentTopics = 30

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 10 seconds

	// Detect existing content topics from previous test
	if len(s.contentFilter.PubsubTopic) > 0 {
		existingTopics := len(s.contentFilter.ContentTopicsList())
		if existingTopics > 0 {
			maxContentTopics = maxContentTopics - existingTopics
		}
	}

	messages := prepareData(maxContentTopics+1, false, true, true)

	// Subscribe
	for _, m := range messages[:len(messages)-1] {
		s.subDetails = s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// All messages within limit should get received
	s.waitForMessages(func() {
		s.publishMessages(messages[:len(messages)-1])
	}, s.subDetails, messages[:len(messages)-1])

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
	s.log = log
	s.wg = &sync.WaitGroup{}

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds

	fullNodeIDHex := make([]byte, hex.EncodedLen(len([]byte(s.fullNodeHost.ID()))))
	_ = hex.Encode(fullNodeIDHex, []byte(s.fullNodeHost.ID()))

	s.log.Info("Already subscribed to", zap.String("fullNode", string(fullNodeIDHex)))

	// This will overwrite values with the second node info
	s.relayNode, s.fullNode = s.makeWakuFilterFullNode(s.testTopic, false)

	// Connect to second full and relay node
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)
	err := s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	fullNodeIDHex = make([]byte, hex.EncodedLen(len([]byte(s.fullNodeHost.ID()))))
	_ = hex.Encode(fullNodeIDHex, []byte(s.fullNodeHost.ID()))

	s.log.Info("Subscribing to second", zap.String("fullNode", string(fullNodeIDHex)))

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

	lightNode2 := s.makeWakuFilterLightNode(true, true)

	// Connect node2
	lightNode2.h.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)

	messages := prepareData(2, true, true, true)

	// Subscribe separately: light node 1 -> full node
	contentFilter := protocol.ContentFilter{PubsubTopic: messages[0].pubSubTopic, ContentTopics: protocol.NewContentTopicSet(messages[0].contentTopic)}
	_, err := s.lightNode.Subscribe(s.ctx, contentFilter, WithPeer(s.fullNodeHost.ID()))
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
