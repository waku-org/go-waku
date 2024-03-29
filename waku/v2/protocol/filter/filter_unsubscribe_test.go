package filter

import (
	"context"
	"crypto/rand"
	"strconv"
	"time"

	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"go.uber.org/zap"
)

func (s *FilterTestSuite) TestUnsubscribeSingleContentTopic() {

	var newContentTopic = "TopicB"

	// Initial subscribe
	s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())
	s.subscribe(s.testTopic, newContentTopic, s.fullNodeHost.ID())

	// Message is possible to receive for original contentTopic
	s.waitForMsg(&WakuMsg{s.testTopic, s.testContentTopic, "test_msg"})

	// Message is possible to receive for new contentTopic
	s.waitForMsg(&WakuMsg{s.testTopic, newContentTopic, "test_msg"})

	_ = s.unsubscribe(s.testTopic, newContentTopic, s.fullNodeHost.ID())

	// Message should not be received for new contentTopic as it was unsubscribed
	s.waitForTimeout(&WakuMsg{s.testTopic, newContentTopic, "test_msg"})

	// Message is still possible to receive for original contentTopic
	s.waitForMsg(&WakuMsg{s.testTopic, s.testContentTopic, "test_msg2"})

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestUnsubscribeMultiContentTopic() {

	var messages = prepareData(3, false, true, true, nil)

	// Subscribe with 3 content topics
	for _, m := range messages {
		s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
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
		s.waitForTimeout(&WakuMsg{m.pubSubTopic, m.contentTopic, m.payload})
	}

	// Message is still possible to receive for the first contentTopic
	s.waitForMsg(&WakuMsg{messages[0].pubSubTopic, messages[0].contentTopic, messages[0].payload})

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestUnsubscribeMultiPubSubMultiContentTopic() {

	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second)

	s.MakeWakuFilterLightNode()
	s.StartLightNode()

	s.MakeWakuFilterFullNode(s.testTopic, true)

	// Connect nodes
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNode.h), peerstore.PermanentAddrTTL)
	err := s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	messages := prepareData(2, true, true, true, nil)

	// Subscribe
	for _, m := range messages {
		s.subDetails = append(s.subDetails, s.getSub(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())...)
		_, err = s.relayNode.Subscribe(context.Background(), protocol.NewContentFilter(m.pubSubTopic))
		s.Require().NoError(err)
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	// Unsubscribe
	for _, m := range messages {
		_ = s.unsubscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// No messages can be received with previous subscriptions
	for i, m := range messages {
		s.waitForTimeoutFromChan(&WakuMsg{m.pubSubTopic, m.contentTopic, m.payload}, s.subDetails[i].C)
	}
}

func (s *FilterTestSuite) TestUnsubscribeErrorHandling() {

	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second)

	s.MakeWakuFilterLightNode()
	s.StartLightNode()

	s.MakeWakuFilterFullNode(s.testTopic, true)

	// Connect nodes
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)
	err := s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	var messages, invalidMessages []WakuMsg

	messages = prepareData(2, false, true, true, nil)

	// Prepare "invalid" data for unsubscribe
	invalidMessages = append(invalidMessages,
		WakuMsg{
			pubSubTopic:  "",
			contentTopic: messages[0].contentTopic,
			payload:      "N/A",
		},
		WakuMsg{
			pubSubTopic:  messages[0].pubSubTopic,
			contentTopic: "",
			payload:      "N/A",
		},
		WakuMsg{
			pubSubTopic:  "/waku/2/go/filter/not_subscribed",
			contentTopic: "not_subscribed_topic",
			payload:      "N/A",
		})

	// Subscribe with valid topics
	for _, m := range messages {
		s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
		_, err = s.relayNode.Subscribe(context.Background(), protocol.NewContentFilter(m.pubSubTopic))
		s.Require().NoError(err)
	}

	// All messages should be possible to receive for subscribed topics
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	// Unsubscribe with empty pubsub
	contentFilter := protocol.ContentFilter{PubsubTopic: invalidMessages[0].pubSubTopic,
		ContentTopics: protocol.NewContentTopicSet(invalidMessages[0].contentTopic)}
	_, err = s.lightNode.Unsubscribe(s.ctx, contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().Error(err)

	// Unsubscribe with empty content topic
	contentFilter = protocol.ContentFilter{PubsubTopic: invalidMessages[1].pubSubTopic,
		ContentTopics: protocol.NewContentTopicSet(invalidMessages[1].contentTopic)}
	_, err = s.lightNode.Unsubscribe(s.ctx, contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().Error(err)

	// Unsubscribe with non-existent topics, expect no error to prevent attacker from topic guessing
	contentFilter = protocol.ContentFilter{PubsubTopic: invalidMessages[2].pubSubTopic,
		ContentTopics: protocol.NewContentTopicSet(invalidMessages[2].contentTopic)}
	_, err = s.lightNode.Unsubscribe(s.ctx, contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	// All messages should be still possible to receive for subscribed topics
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestUnsubscribeAllWithoutContentTopics() {

	var messages = prepareData(2, false, true, true, nil)

	// Subscribe with 2 content topics
	for _, m := range messages {
		s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	// Unsubscribe all with peer specification
	_, err := s.lightNode.UnsubscribeAll(s.ctx, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	// Messages should not be received for any contentTopics
	for _, m := range messages {
		s.waitForTimeout(&WakuMsg{m.pubSubTopic, m.contentTopic, m.payload})
	}
}

func (s *FilterTestSuite) TestUnsubscribeAllDiffPubSubContentTopics() {

	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second)

	s.MakeWakuFilterLightNode()
	s.StartLightNode()

	s.MakeWakuFilterFullNode(s.testTopic, true)

	// Connect nodes
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNode.h), peerstore.PermanentAddrTTL)
	err := s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	messages := prepareData(2, true, true, true, nil)

	// Subscribe
	for _, m := range messages {
		s.subDetails = append(s.subDetails, s.getSub(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())...)
		_, err = s.relayNode.Subscribe(context.Background(), protocol.NewContentFilter(m.pubSubTopic))
		s.Require().NoError(err)
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	// Unsubscribe all without any specification
	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

	// No messages can be received with previous subscriptions
	for i, m := range messages {
		s.waitForTimeoutFromChan(&WakuMsg{m.pubSubTopic, m.contentTopic, m.payload}, s.subDetails[i].C)
	}

}

func (s *FilterTestSuite) TestUnsubscribeAllUnrelatedPeer() {

	var messages = prepareData(2, false, true, false, nil)

	// Subscribe with 2 content topics
	for _, m := range messages {
		s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	// Create new host - not related to any node
	host, err := tests.MakeHost(context.Background(), 12345, rand.Reader)
	s.Require().NoError(err)

	s.Log.Info("Host ID", logging.HostID("FullNode", s.fullNodeHost.ID()))
	s.Log.Info("Host ID", logging.HostID("LightNode", s.lightNodeHost.ID()))
	s.Log.Info("Host ID", logging.HostID("Unrelated", host.ID()))

	// Unsubscribe all with unrelated peer specification
	pushResult, err := s.lightNode.UnsubscribeAll(s.ctx, WithPeer(host.ID()))

	for e := range pushResult.errs {
		s.Log.Info("Push Result ", zap.String("error", strconv.Itoa(e)))
	}

	// All messages should be received because peer ID used was not related to any subscription
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	// Expect error for unsubscribe from non existing peer
	s.Require().Error(err)
}
