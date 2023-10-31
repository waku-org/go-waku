package filter

import (
	"context"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"time"
)

func (s *FilterTestSuite) TestUnsubscribeSingleContentTopic() {

	var newContentTopic = "TopicB"

	// Initial subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())
	s.subDetails = s.subscribe(s.testTopic, newContentTopic, s.fullNodeHost.ID())

	// Message is possible to receive for original contentTopic
	s.waitForMsg(func() {
		s.publishMsg(s.testTopic, s.testContentTopic, "test_msg")
	}, s.subDetails[0].C, false)

	// Message is possible to receive for new contentTopic
	s.waitForMsg(func() {
		s.publishMsg(s.testTopic, newContentTopic, "test_msg")
	}, s.subDetails[0].C, false)

	_ = s.unsubscribe(s.testTopic, newContentTopic, s.fullNodeHost.ID())

	// Message should not be received for new contentTopic as it was unsubscribed
	s.waitForMsg(func() {
		s.publishMsg(s.testTopic, newContentTopic, "test_msg")
	}, s.subDetails[0].C, true)

	// Message is still possible to receive for original contentTopic
	s.waitForMsg(func() {
		s.publishMsg(s.testTopic, s.testContentTopic, "test_msg2")
	}, s.subDetails[0].C, false)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestUnsubscribeMultiContentTopic() {

	var messages = prepareData(3, false, true, true)

	// Subscribe with 3 content topics
	for _, m := range messages {
		s.subDetails = s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	// Unsubscribe with the last 2 content topics
	for _, m := range messages[1:] {
		_ = s.unsubscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// Messages should not be received for the last two contentTopics as it was unsubscribed
	for _, m := range messages[1:] {
		s.waitForMsg(func() {
			s.publishMsg(m.pubSubTopic, m.contentTopic, m.payload)
		}, s.subDetails[0].C, true)
	}

	// Message is still possible to receive for the first contentTopic
	s.waitForMsg(func() {
		s.publishMsg(messages[0].pubSubTopic, messages[0].contentTopic, messages[0].payload)
	}, s.subDetails[0].C, false)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestUnsubscribeMultiPubSubMultiContentTopic() {
	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 20 seconds

	s.lightNode = s.makeWakuFilterLightNode(true, true)

	s.relayNode, s.fullNode = s.makeWakuFilterFullNode(s.testTopic, true, true)

	// Connect nodes
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)
	err := s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	messages := prepareData(2, true, true, true)

	// Subscribe
	for _, m := range messages {
		s.subDetails = append(s.subDetails, s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())...)
		_, err = s.relayNode.Subscribe(context.Background(), protocol.NewContentFilter(m.pubSubTopic))
		s.Require().NoError(err)
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	// Unsubscribe
	for _, m := range messages {
		s.subDetails = s.unsubscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// No messages can be sent or received without any subscription
	s.Require().Equal(len(s.subDetails), 0, "Number of subscriptions is not 0")

}
