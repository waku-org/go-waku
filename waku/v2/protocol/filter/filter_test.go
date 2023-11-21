package filter

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/suite"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"
	"github.com/waku-org/go-waku/waku/v2/service"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

func TestFilterSuite(t *testing.T) {
	suite.Run(t, new(FilterTestSuite))
}

type FilterTestSuite struct {
	suite.Suite

	testTopic        string
	testContentTopic string
	ctx              context.Context
	ctxCancel        context.CancelFunc
	lightNode        *WakuFilterLightNode
	lightNodeHost    host.Host
	relayNode        *relay.WakuRelay
	relaySub         *relay.Subscription
	fullNode         *WakuFilterFullNode
	fullNodeHost     host.Host
	wg               *sync.WaitGroup
	contentFilter    protocol.ContentFilter
	subDetails       []*subscription.SubscriptionDetails
	log              *zap.Logger
}

type WakuMsg struct {
	pubSubTopic  string
	contentTopic string
	payload      string
}

func (s *FilterTestSuite) makeWakuRelay(topic string, shared bool) (*relay.WakuRelay, *relay.Subscription, host.Host, relay.Broadcaster) {

	broadcaster := relay.NewBroadcaster(10)
	s.Require().NoError(broadcaster.Start(context.Background()))

	port, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	s.Require().NoError(err)

	relay := relay.NewWakuRelay(broadcaster, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, s.log)
	relay.SetHost(host)

	if shared {
		s.fullNodeHost = host
	}

	err = relay.Start(context.Background())
	s.Require().NoError(err)

	sub, err := relay.Subscribe(context.Background(), protocol.NewContentFilter(topic))
	s.Require().NoError(err)

	return relay, sub[0], host, broadcaster
}

func (s *FilterTestSuite) makeWakuFilterLightNode(start bool, withBroadcaster bool) *WakuFilterLightNode {
	port, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	s.Require().NoError(err)
	var b relay.Broadcaster
	if withBroadcaster {
		b = relay.NewBroadcaster(10)
		s.Require().NoError(b.Start(context.Background()))
	}
	filterPush := NewWakuFilterLightNode(b, nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, s.log)
	filterPush.SetHost(host)
	s.lightNodeHost = host
	if start {
		err = filterPush.Start(context.Background())
		s.Require().NoError(err)
	}

	return filterPush
}

func (s *FilterTestSuite) makeWakuFilterFullNode(topic string, withRegisterAll bool, shared bool) (*relay.WakuRelay, *WakuFilterFullNode) {
	var sub *relay.Subscription
	node, relaySub, host, broadcaster := s.makeWakuRelay(topic, shared)

	if shared {
		s.relaySub = relaySub
	}

	node2Filter := NewWakuFilterFullNode(timesource.NewDefaultClock(), prometheus.DefaultRegisterer, s.log)
	node2Filter.SetHost(host)

	if withRegisterAll {
		sub = broadcaster.RegisterForAll()
	} else {
		sub = broadcaster.Register(protocol.NewContentFilter(topic))
	}

	err := node2Filter.Start(s.ctx, sub)
	s.Require().NoError(err)

	return node, node2Filter
}

func (s *FilterTestSuite) waitForMsg(fn func(), ch chan *protocol.Envelope) {
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

	if fn != nil {
		fn()
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
	s.log.Info("Expected messages ", zap.String("count", strconv.Itoa(msgCount)))
	s.log.Info("Existing subscriptions ", zap.String("count", strconv.Itoa(len(subs))))

	go func() {
		defer s.wg.Done()
		for _, sub := range subs {
			s.log.Info("Looking at ", zap.String("pubSubTopic", sub.ContentFilter.PubsubTopic))
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
					s.log.Info("received message ", zap.String("pubSubTopic", received.pubSubTopic), zap.String("contentTopic", received.contentTopic), zap.String("payload", received.payload))
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

func (s *FilterTestSuite) waitForTimeout(fn func(), ch chan *protocol.Envelope) {
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

	fn()

	s.wg.Wait()
}

func (s *FilterTestSuite) subscribe(pubsubTopic string, contentTopic string, peer peer.ID) []*subscription.SubscriptionDetails {

	for _, sub := range s.subDetails {
		if sub.ContentFilter.PubsubTopic == pubsubTopic {
			sub.Add(contentTopic)
			s.contentFilter = sub.ContentFilter
			subDetails, err := s.lightNode.Subscribe(s.ctx, s.contentFilter, WithPeer(peer))
			s.Require().NoError(err)
			return subDetails
		}
	}

	s.contentFilter = protocol.ContentFilter{PubsubTopic: pubsubTopic, ContentTopics: protocol.NewContentTopicSet(contentTopic)}

	subDetails, err := s.lightNode.Subscribe(s.ctx, s.contentFilter, WithPeer(peer))
	s.Require().NoError(err)

	// Sleep to make sure the filter is subscribed
	time.Sleep(1 * time.Second)

	return subDetails
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

func (s *FilterTestSuite) publishMsg(topic, contentTopic string, optionalPayload ...string) {
	var payload string
	if len(optionalPayload) > 0 {
		payload = optionalPayload[0]
	} else {
		payload = "123"
	}

	_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage(contentTopic, utils.GetUnixEpoch(), payload), relay.WithPubSubTopic(topic))
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
		pubsubTopic     = "/waku/2/go/filter/test" // Has to be the same with initial s.testTopic
		contentTopic    = "/test/10/my-app"        // Has to be the same with initial s.testContentTopic
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

func (s *FilterTestSuite) SetupTest() {
	log := utils.Logger() //.Named("filterv2-test")
	s.log = log
	// Use a pointer to WaitGroup so that to avoid copying
	// https://pkg.go.dev/sync#WaitGroup
	s.wg = &sync.WaitGroup{}

	// Create test context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds
	s.ctx = ctx
	s.ctxCancel = cancel

	s.testTopic = "/waku/2/go/filter/test"
	s.testContentTopic = "/test/10/my-app"

	s.lightNode = s.makeWakuFilterLightNode(true, true)

	//TODO: Add tests to verify broadcaster.

	s.relayNode, s.fullNode = s.makeWakuFilterFullNode(s.testTopic, false, true)

	// Connect nodes
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)
	err := s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TearDownTest() {
	s.fullNode.Stop()
	s.relayNode.Stop()
	s.relaySub.Unsubscribe()
	s.lightNode.Stop()
	s.ctxCancel()
}

func (s *FilterTestSuite) TestPeerFailure() {
	broadcaster2 := relay.NewBroadcaster(10)
	s.Require().NoError(broadcaster2.Start(context.Background()))

	// Initial subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Simulate there's been a failure before
	s.fullNode.subscriptions.FlagAsFailure(s.lightNodeHost.ID())

	// Sleep to make sure the filter is subscribed
	time.Sleep(2 * time.Second)

	s.Require().True(s.fullNode.subscriptions.IsFailedPeer(s.lightNodeHost.ID()))

	s.waitForMsg(func() {
		s.publishMsg(s.testTopic, s.testContentTopic)
	}, s.subDetails[0].C)

	// Failure is removed
	s.Require().False(s.fullNode.subscriptions.IsFailedPeer(s.lightNodeHost.ID()))

	// Kill the subscriber
	s.lightNodeHost.Close()

	time.Sleep(1 * time.Second)

	s.publishMsg(s.testTopic, s.testContentTopic)

	// TODO: find out how to eliminate this sleep
	time.Sleep(1 * time.Second)
	s.Require().True(s.fullNode.subscriptions.IsFailedPeer(s.lightNodeHost.ID()))

	time.Sleep(2 * time.Second)

	s.publishMsg(s.testTopic, s.testContentTopic)

	time.Sleep(2 * time.Second)

	s.Require().True(s.fullNode.subscriptions.IsFailedPeer(s.lightNodeHost.ID())) // Failed peer has been removed
	s.Require().False(s.fullNode.subscriptions.Has(s.lightNodeHost.ID()))         // Failed peer has been removed
}

func (s *FilterTestSuite) TestRunningGuard() {

	s.lightNode.Stop()

	contentFilter := protocol.ContentFilter{PubsubTopic: "test", ContentTopics: protocol.NewContentTopicSet("test")}

	_, err := s.lightNode.Subscribe(s.ctx, contentFilter, WithPeer(s.fullNodeHost.ID()))

	s.Require().ErrorIs(err, service.ErrNotStarted)

	err = s.lightNode.Start(s.ctx)
	s.Require().NoError(err)

	_, err = s.lightNode.Subscribe(s.ctx, contentFilter, WithPeer(s.fullNodeHost.ID()))

	s.Require().NoError(err)
}

func (s *FilterTestSuite) TestFireAndForgetAndCustomWg() {

	contentFilter := protocol.ContentFilter{PubsubTopic: "test", ContentTopics: protocol.NewContentTopicSet("test")}

	_, err := s.lightNode.Subscribe(s.ctx, contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	result, err := s.lightNode.Unsubscribe(s.ctx, contentFilter, DontWait())
	s.Require().NoError(err)
	s.Require().Equal(0, len(result.Errors()))

	_, err = s.lightNode.Subscribe(s.ctx, contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	wg := sync.WaitGroup{}
	_, err = s.lightNode.Unsubscribe(s.ctx, contentFilter, WithWaitGroup(&wg))
	wg.Wait()
	s.Require().NoError(err)
}

func (s *FilterTestSuite) TestStartStop() {

	var wg sync.WaitGroup
	wg.Add(2)
	s.lightNode = s.makeWakuFilterLightNode(false, false)

	stopNode := func() {
		for i := 0; i < 100000; i++ {
			s.lightNode.Stop()
		}
		wg.Done()
	}

	startNode := func() {
		for i := 0; i < 100; i++ {
			err := s.lightNode.Start(context.Background())
			if errors.Is(err, service.ErrAlreadyStarted) {
				continue
			}
			s.Require().NoError(err)
		}
		wg.Done()
	}

	go startNode()
	go stopNode()

	wg.Wait()
}

func (s *FilterTestSuite) TestAutoShard() {

	//Workaround as could not find a way to reuse setup test with params
	// Stop what is run in setup
	s.fullNode.Stop()
	s.lightNode.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 10 seconds
	s.ctx = ctx
	s.ctxCancel = cancel

	cTopic1Str := "0/test/1/testTopic/proto"
	cTopic1, err := protocol.StringToContentTopic(cTopic1Str)
	s.Require().NoError(err)
	//Computing pubSubTopic only for filterFullNode.
	pubSubTopic := protocol.GetShardFromContentTopic(cTopic1, protocol.GenerationZeroShardsCount)
	s.testContentTopic = cTopic1Str
	s.testTopic = pubSubTopic.String()

	s.lightNode = s.makeWakuFilterLightNode(true, false)
	s.relayNode, s.fullNode = s.makeWakuFilterFullNode(pubSubTopic.String(), false, true)

	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)
	err = s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	s.log.Info("Testing Autoshard:CreateSubscription")
	s.subDetails = s.subscribe("", s.testContentTopic, s.fullNodeHost.ID())
	s.waitForMsg(func() {
		_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch()), relay.WithPubSubTopic(s.testTopic))
		s.Require().NoError(err)

	}, s.subDetails[0].C)

	// Wrong content topic
	s.waitForTimeout(func() {
		s.publishMsg(s.testTopic, "TopicB", "second")
	}, s.subDetails[0].C)

	_, err = s.lightNode.Unsubscribe(s.ctx, s.contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	time.Sleep(1 * time.Second)

	// Should not receive after unsubscribe
	s.waitForTimeout(func() {
		s.publishMsg(s.testTopic, s.testContentTopic, "third")
	}, s.subDetails[0].C)

	s.subDetails = s.subscribe("", s.testContentTopic, s.fullNodeHost.ID())

	s.log.Info("Testing Autoshard:SubscriptionPing")
	err = s.lightNode.Ping(context.Background(), s.fullNodeHost.ID())
	s.Require().NoError(err)

	// Test ModifySubscription Subscribe to another content_topic
	s.log.Info("Testing Autoshard:ModifySubscription")

	newContentTopic := "0/test/1/testTopic1/proto"
	s.subDetails = s.subscribe("", newContentTopic, s.fullNodeHost.ID())

	s.waitForMsg(func() {
		_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage(newContentTopic, utils.GetUnixEpoch()), relay.WithPubSubTopic(s.testTopic))
		s.Require().NoError(err)

	}, s.subDetails[0].C)

	_, err = s.lightNode.Unsubscribe(s.ctx, protocol.ContentFilter{
		PubsubTopic:   s.testTopic,
		ContentTopics: protocol.NewContentTopicSet(newContentTopic),
	})
	s.Require().NoError(err)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) BeforeTest(suiteName, testName string) {
	s.log.Info("Executing ", zap.String("testName", testName))
}

func (s *FilterTestSuite) AfterTest(suiteName, testName string) {
	s.log.Info("Finished executing ", zap.String("testName", testName))
}
