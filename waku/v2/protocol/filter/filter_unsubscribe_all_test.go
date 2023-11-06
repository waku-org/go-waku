package filter

import (
	"context"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"time"
)

func (s *FilterTestSuite) TestUnsubscribeAllWithoutContentTopics() {

	var messages = prepareData(2, false, true, true)

	// Subscribe with 2 content topics
	for _, m := range messages {
		s.subDetails = s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
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
		s.waitForTimeout(func() {
			s.publishMsg(m.pubSubTopic, m.contentTopic, m.payload)
		}, s.subDetails[0].C)
	}
}

func (s *FilterTestSuite) TestUnsubscribeAllDiffPubSubContentTopics() {

	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second)

	s.lightNode = s.makeWakuFilterLightNode(true, true)

	s.relayNode, s.fullNode = s.makeWakuFilterFullNode(s.testTopic, true, true)

	// Connect nodes
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNode.h), peerstore.PermanentAddrTTL)
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

	// Unsubscribe all without any specification
	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

	// No messages can be received with previous subscriptions
	for i, m := range messages {
		s.waitForTimeout(func() {
			s.publishMsg(m.pubSubTopic, m.contentTopic, m.payload)
		}, s.subDetails[i].C)
	}

}
