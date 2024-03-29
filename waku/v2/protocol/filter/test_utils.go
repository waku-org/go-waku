package filter

import (
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/suite"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

type LightNodeData struct {
	lightNode     *WakuFilterLightNode
	lightNodeHost host.Host
}

type FullNodeData struct {
	relayNode    *relay.WakuRelay
	RelaySub     *relay.Subscription
	fullNodeHost host.Host
	Broadcaster  relay.Broadcaster
	fullNode     *WakuFilterFullNode
}

type FilterTestSuite struct {
	suite.Suite
	LightNodeData
	FullNodeData

	testTopic        string
	testContentTopic string
	ctx              context.Context
	ctxCancel        context.CancelFunc
	wg               *sync.WaitGroup
	contentFilter    protocol.ContentFilter
	subDetails       []*subscription.SubscriptionDetails

	Log *zap.Logger
}

type WakuMsg struct {
	pubSubTopic  string
	contentTopic string
	payload      string
}

func (s *FilterTestSuite) GetWakuRelay(topic string) FullNodeData {

	broadcaster := relay.NewBroadcaster(10)
	s.Require().NoError(broadcaster.Start(context.Background()))

	port, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	s.Require().NoError(err)

	relay := relay.NewWakuRelay(broadcaster, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, s.Log)
	relay.SetHost(host)

	err = relay.Start(context.Background())
	s.Require().NoError(err)

	sub, err := relay.Subscribe(context.Background(), protocol.NewContentFilter(topic))
	s.Require().NoError(err)

	return FullNodeData{relay, sub[0], host, broadcaster, nil}
}

func (s *FilterTestSuite) GetWakuFilterFullNode(topic string, withRegisterAll bool) FullNodeData {

	nodeData := s.GetWakuRelay(topic)

	node2Filter := NewWakuFilterFullNode(timesource.NewDefaultClock(), prometheus.DefaultRegisterer, s.Log)
	node2Filter.SetHost(nodeData.fullNodeHost)

	var sub *relay.Subscription
	if withRegisterAll {
		sub = nodeData.Broadcaster.RegisterForAll()
	} else {
		sub = nodeData.Broadcaster.Register(protocol.NewContentFilter(topic))
	}

	err := node2Filter.Start(s.ctx, sub)
	s.Require().NoError(err)

	nodeData.fullNode = node2Filter

	return nodeData
}

func (s *FilterTestSuite) MakeWakuFilterFullNode(topic string, withRegisterAll bool) {
	nodeData := s.GetWakuFilterFullNode(topic, withRegisterAll)

	s.FullNodeData = nodeData
}

func (s *FilterTestSuite) GetWakuFilterLightNode() LightNodeData {
	port, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	s.Require().NoError(err)
	b := relay.NewBroadcaster(10)
	s.Require().NoError(b.Start(context.Background()))
	filterPush := NewWakuFilterLightNode(b, nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, s.Log)
	filterPush.SetHost(host)

	return LightNodeData{filterPush, host}
}

func (s *FilterTestSuite) MakeWakuFilterLightNode() {
	s.LightNodeData = s.GetWakuFilterLightNode()
}

func (s *FilterTestSuite) StartLightNode() {
	err := s.lightNode.Start(context.Background())
	s.Require().NoError(err)
}

func (s *FilterTestSuite) waitForMsg(msg *WakuMsg) {
	s.waitForMsgFromChan(msg, s.subDetails[0].C)
}

func (s *FilterTestSuite) waitForMsgFromChan(msg *WakuMsg, ch chan *protocol.Envelope) {
	s.wg.Add(1)
	var msgFound = false
	go func() {
		defer s.wg.Done()
		select {
		case env := <-ch:
			for _, topic := range s.contentFilter.ContentTopicsList() {
				if topic == env.Message().GetContentTopic() {
					msgFound = true
				}
			}
			s.Require().True(msgFound)
		case <-time.After(1 * time.Second):
			s.Require().Fail("Message timeout")
		case <-s.ctx.Done():
			s.Require().Fail("test exceeded allocated time")
		}
	}()

	if msg != nil {
		s.publishMsg(msg)
	}

	s.wg.Wait()
}

func matchOneOfManyMsg(one WakuMsg, many []WakuMsg) bool {
	for _, m := range many {
		if m.pubSubTopic == one.pubSubTopic &&
			m.contentTopic == one.contentTopic &&
			m.payload == one.payload {
			return true
		}
	}

	return false
}

func (s *FilterTestSuite) waitForMessages(fn func(), subs []*subscription.SubscriptionDetails, expected []WakuMsg) {
	s.wg.Add(1)
	msgCount := len(expected)
	found := 0
	s.Log.Info("Expected messages ", zap.String("count", strconv.Itoa(msgCount)))
	s.Log.Info("Existing subscriptions ", zap.String("count", strconv.Itoa(len(subs))))

	go func() {
		defer s.wg.Done()
		for _, sub := range subs {
			s.Log.Info("Looking at ", zap.String("pubSubTopic", sub.ContentFilter.PubsubTopic))
			for i := 0; i < msgCount; i++ {
				select {
				case env, ok := <-sub.C:
					if !ok {
						continue
					}
					received := WakuMsg{
						pubSubTopic:  env.PubsubTopic(),
						contentTopic: env.Message().GetContentTopic(),
						payload:      string(env.Message().GetPayload()),
					}
					s.Log.Debug("received message ", zap.String("pubSubTopic", received.pubSubTopic), zap.String("contentTopic", received.contentTopic), zap.String("payload", received.payload))
					if matchOneOfManyMsg(received, expected) {
						found++
					}
				case <-time.After(3 * time.Second):

				case <-s.ctx.Done():
					s.Require().Fail("test exceeded allocated time")
				}
			}
		}
	}()

	if fn != nil {
		fn()
	}

	s.wg.Wait()
	s.Require().True(msgCount == found)
}

func (s *FilterTestSuite) waitForTimeout(msg *WakuMsg) {
	s.waitForTimeoutFromChan(msg, s.subDetails[0].C)
}

func (s *FilterTestSuite) waitForTimeoutFromChan(msg *WakuMsg, ch chan *protocol.Envelope) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		select {
		case env, ok := <-ch:
			if ok {
				s.Require().Fail("should not receive another message", zap.String("payload", string(env.Message().Payload)))
			}
		case <-time.After(1 * time.Second):
			// Timeout elapsed, all good
		case <-s.ctx.Done():
			s.Require().Fail("waitForTimeout test exceeded allocated time")
		}
	}()

	s.publishMsg(msg)

	s.wg.Wait()
}

func (s *FilterTestSuite) getSub(pubsubTopic string, contentTopic string, peer peer.ID) []*subscription.SubscriptionDetails {
	contentFilter := protocol.ContentFilter{PubsubTopic: pubsubTopic, ContentTopics: protocol.NewContentTopicSet(contentTopic)}

	subDetails, err := s.lightNode.Subscribe(s.ctx, contentFilter, WithPeer(peer))
	s.Require().NoError(err)

	time.Sleep(1 * time.Second)

	return subDetails
}
func (s *FilterTestSuite) subscribe(pubsubTopic string, contentTopic string, peer peer.ID) {

	for _, sub := range s.subDetails {
		if sub.ContentFilter.PubsubTopic == pubsubTopic {
			sub.Add(contentTopic)
			s.contentFilter = sub.ContentFilter
			subDetails, err := s.lightNode.Subscribe(s.ctx, s.contentFilter, WithPeer(peer))
			s.subDetails = subDetails
			s.Require().NoError(err)
			return
		}
	}

	s.subDetails = s.getSub(pubsubTopic, contentTopic, peer)
	s.contentFilter = s.subDetails[0].ContentFilter
}

func (s *FilterTestSuite) unsubscribe(pubsubTopic string, contentTopic string, peer peer.ID) []*subscription.SubscriptionDetails {

	for _, sub := range s.subDetails {
		if sub.ContentFilter.PubsubTopic == pubsubTopic {
			topicsCount := len(sub.ContentFilter.ContentTopicsList())
			if topicsCount == 1 {
				_, err := s.lightNode.Unsubscribe(s.ctx, sub.ContentFilter, WithPeer(peer))
				s.Require().NoError(err)
			} else {
				sub.Remove(contentTopic)
			}
			s.contentFilter = sub.ContentFilter
		}
	}

	return s.lightNode.Subscriptions()
}

func (s *FilterTestSuite) publishMsg(msg *WakuMsg) {
	if len(msg.payload) == 0 {
		msg.payload = "123"
	}

	_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage(msg.contentTopic, utils.GetUnixEpoch(), msg.payload), relay.WithPubSubTopic(msg.pubSubTopic))
	s.Require().NoError(err)
}

func (s *FilterTestSuite) publishMessages(msgs []WakuMsg) {
	for _, m := range msgs {
		_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage(m.contentTopic, utils.GetUnixEpoch(), m.payload), relay.WithPubSubTopic(m.pubSubTopic))
		s.Require().NoError(err)
	}
}

func prepareData(quantity int, topics, contentTopics, payloads bool, sg tests.StringGenerator) []WakuMsg {
	var (
		pubsubTopic     = defaultTestPubSubTopic  // Has to be the same with initial s.testTopic
		contentTopic    = defaultTestContentTopic // Has to be the same with initial s.testContentTopic
		payload         = "test_msg"
		messages        []WakuMsg
		strMaxLenght    = 4097
		generatedString = ""
	)

	for i := 0; i < quantity; i++ {
		msg := WakuMsg{
			pubSubTopic:  pubsubTopic,
			contentTopic: contentTopic,
			payload:      payload,
		}

		if sg != nil {
			generatedString, _ = sg(strMaxLenght)

		}

		if topics {
			msg.pubSubTopic = fmt.Sprintf("%s%02d%s", pubsubTopic, i, generatedString)
		}

		if contentTopics {
			msg.contentTopic = fmt.Sprintf("%s%02d%s", contentTopic, i, generatedString)
		}

		if payloads {
			msg.payload = fmt.Sprintf("%s%02d%s", payload, i, generatedString)
		}

		messages = append(messages, msg)
	}

	return messages
}
