package filter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
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

func (s *FilterTestSuite) makeWakuRelay(topic string) (*relay.WakuRelay, *relay.Subscription, host.Host, relay.Broadcaster) {

	broadcaster := relay.NewBroadcaster(10)
	s.Require().NoError(broadcaster.Start(context.Background()))

	port, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	s.Require().NoError(err)

	relay := relay.NewWakuRelay(broadcaster, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, s.log)
	relay.SetHost(host)
	s.fullNodeHost = host
	err = relay.Start(context.Background())
	s.Require().NoError(err)

	sub, err := relay.SubscribeToTopic(context.Background(), topic)
	s.Require().NoError(err)

	return relay, sub, host, broadcaster
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

func (s *FilterTestSuite) makeWakuFilterFullNode(topic string) (*relay.WakuRelay, *WakuFilterFullNode) {
	node, relaySub, host, broadcaster := s.makeWakuRelay(topic)
	s.relaySub = relaySub

	node2Filter := NewWakuFilterFullNode(timesource.NewDefaultClock(), prometheus.DefaultRegisterer, s.log)
	node2Filter.SetHost(host)
	sub := broadcaster.RegisterForAll()
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
		case <-time.After(5 * time.Second):
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
				case env := <-sub.C:
					received := WakuMsg{
						pubSubTopic:  env.PubsubTopic(),
						contentTopic: env.Message().GetContentTopic(),
						payload:      string(env.Message().GetPayload()),
					}
					s.log.Info("received message ", zap.String("pubSubTopic", received.pubSubTopic),  zap.String("contentTopic", received.contentTopic), zap.String("payload", received.payload))
					if matchOneOfManyMsg(received, expected) {
						found++
					}
				case <-time.After(2 * time.Second):

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

func (s *FilterTestSuite) unsubscribe(pubsubTopic string, contentTopic string, peer peer.ID) <-chan WakuFilterPushResult {

	for _, sub := range s.subDetails {
		if sub.ContentFilter.PubsubTopic == pubsubTopic {
			topicsCount := len(s.contentFilter.ContentTopicsList())
			if topicsCount == 1 {
				_, err := s.lightNode.Unsubscribe(s.ctx, sub.ContentFilter, WithPeer(peer))
				s.Require().NoError(err)
			} else {
				sub.Remove(contentTopic)
			}
			s.contentFilter = sub.ContentFilter
			return nil
		}
	}

	return nil
}

func (s *FilterTestSuite) publishMsg(topic, contentTopic string, optionalPayload ...string) {
	var payload string
	if len(optionalPayload) > 0 {
		payload = optionalPayload[0]
	} else {
		payload = "123"
	}

	_, err := s.relayNode.PublishToTopic(s.ctx, tests.CreateWakuMessage(contentTopic, utils.GetUnixEpoch(), payload), topic)
	s.Require().NoError(err)
}

func (s *FilterTestSuite) publishMessages(msgs []WakuMsg) {
	for _, m := range msgs {
		_, err := s.relayNode.PublishToTopic(s.ctx, tests.CreateWakuMessage(m.contentTopic, utils.GetUnixEpoch(), m.payload), m.pubSubTopic)
		s.Require().NoError(err)
		//dst := make([]byte, hex.EncodedLen(len(hash)))
		//hex.Encode(dst, hash)
		//s.log.Info("Published message ", zap.String("hash:", string(dst)))
	}
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
	s.testContentTopic = "TopicA"

	s.lightNode = s.makeWakuFilterLightNode(true, true)

	//TODO: Add tests to verify broadcaster.

	s.relayNode, s.fullNode = s.makeWakuFilterFullNode(s.testTopic)

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

func (s *FilterTestSuite) TestSubscriptionPing() {
	err := s.lightNode.Ping(context.Background(), s.fullNodeHost.ID())
	s.Require().Error(err)
	filterErr, ok := err.(*FilterError)
	s.Require().True(ok)
	s.Require().Equal(filterErr.Code, http.StatusNotFound)

	contentTopic := "abc"
	s.subDetails = s.subscribe(s.testTopic, contentTopic, s.fullNodeHost.ID())

	err = s.lightNode.Ping(context.Background(), s.fullNodeHost.ID())
	s.Require().NoError(err)
}

func (s *FilterTestSuite) TestUnSubscriptionPing() {

	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	err := s.lightNode.Ping(context.Background(), s.fullNodeHost.ID())
	s.Require().NoError(err)

	_, err = s.lightNode.Unsubscribe(s.ctx, s.contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	err = s.lightNode.Ping(context.Background(), s.fullNodeHost.ID())
	s.Require().Error(err)
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

func (s *FilterTestSuite) TestCreateSubscription() {
	// Initial subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())
	s.waitForMsg(func() {
		_, err := s.relayNode.PublishToTopic(s.ctx, tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch()), s.testTopic)
		s.Require().NoError(err)

	}, s.subDetails[0].C)
}

func (s *FilterTestSuite) TestModifySubscription() {

	// Initial subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	s.waitForMsg(func() {
		_, err := s.relayNode.PublishToTopic(s.ctx, tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch()), s.testTopic)
		s.Require().NoError(err)

	}, s.subDetails[0].C)

	// Subscribe to another content_topic
	newContentTopic := "Topic_modified"
	s.subDetails = s.subscribe(s.testTopic, newContentTopic, s.fullNodeHost.ID())

	s.waitForMsg(func() {
		_, err := s.relayNode.PublishToTopic(s.ctx, tests.CreateWakuMessage(newContentTopic, utils.GetUnixEpoch()), s.testTopic)
		s.Require().NoError(err)

	}, s.subDetails[0].C)
}

func (s *FilterTestSuite) TestMultipleMessages() {

	// Initial subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	s.waitForMsg(func() {
		_, err := s.relayNode.PublishToTopic(s.ctx, tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch(), "first"), s.testTopic)
		s.Require().NoError(err)

	}, s.subDetails[0].C)

	s.waitForMsg(func() {
		_, err := s.relayNode.PublishToTopic(s.ctx, tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch(), "second"), s.testTopic)
		s.Require().NoError(err)

	}, s.subDetails[0].C)
}

func (s *FilterTestSuite) TestRunningGuard() {

	s.lightNode.Stop()

	contentFilter := protocol.ContentFilter{PubsubTopic: "test", ContentTopics: protocol.NewContentTopicSet("test")}

	_, err := s.lightNode.Subscribe(s.ctx, contentFilter, WithPeer(s.fullNodeHost.ID()))

	s.Require().ErrorIs(err, protocol.ErrNotStarted)

	err = s.lightNode.Start(s.ctx)
	s.Require().NoError(err)

	_, err = s.lightNode.Subscribe(s.ctx, contentFilter, WithPeer(s.fullNodeHost.ID()))

	s.Require().NoError(err)
}

func (s *FilterTestSuite) TestFireAndForgetAndCustomWg() {

	contentFilter := protocol.ContentFilter{PubsubTopic: "test", ContentTopics: protocol.NewContentTopicSet("test")}

	_, err := s.lightNode.Subscribe(s.ctx, contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	ch, err := s.lightNode.Unsubscribe(s.ctx, contentFilter, DontWait())
	_, open := <-ch
	s.Require().NoError(err)
	s.Require().False(open)

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
			if errors.Is(err, protocol.ErrAlreadyStarted) {
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
	s.relayNode, s.fullNode = s.makeWakuFilterFullNode(pubSubTopic.String())

	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)
	err = s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	s.log.Info("Testing Autoshard:CreateSubscription")
	s.subDetails = s.subscribe("", s.testContentTopic, s.fullNodeHost.ID())
	s.waitForMsg(func() {
		_, err := s.relayNode.PublishToTopic(s.ctx, tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch()), s.testTopic)
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
		_, err := s.relayNode.PublishToTopic(s.ctx, tests.CreateWakuMessage(newContentTopic, utils.GetUnixEpoch()), s.testTopic)
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

func (s *FilterTestSuite) TestIncorrectSubscribeIdentifier() {
	log := utils.Logger()
	s.log = log
	s.wg = &sync.WaitGroup{}

	// Create test context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds
	s.ctx = ctx
	s.ctxCancel = cancel

	s.testTopic = "/waku/2/go/filter/test"
	s.testContentTopic = "TopicA"

	s.lightNode = s.makeWakuFilterLightNode(true, true)

	s.relayNode, s.fullNode = s.makeWakuFilterFullNode(s.testTopic)

	const FilterSubscribeID_Incorrect1 = libp2pProtocol.ID("/vac/waku/filter-subscribe/abcd")

	// Connect nodes
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)
	err := s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_Incorrect1)
	s.Require().NoError(err)

	// Subscribe with incorrect SubscribeID
	s.contentFilter = protocol.ContentFilter{PubsubTopic: s.testTopic, ContentTopics: protocol.NewContentTopicSet(s.testContentTopic)}
	_, err = s.lightNode.Subscribe(s.ctx, s.contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().Error(err)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)
}

func (wf *WakuFilterLightNode) startWithIncorrectPushProto() error {
	const FilterPushID_Incorrect1 = libp2pProtocol.ID("/vac/waku/filter-push/abcd")

	wf.subscriptions = subscription.NewSubscriptionMap(wf.log)
	wf.h.SetStreamHandlerMatch(FilterPushID_v20beta1, protocol.PrefixTextMatch(string(FilterPushID_Incorrect1)), wf.onRequest(wf.Context()))

	wf.log.Info("filter-push incorrect protocol started")
	return nil
}

func (s *FilterTestSuite) TestIncorrectPushIdentifier() {
	log := utils.Logger()
	s.log = log
	s.wg = &sync.WaitGroup{}

	// Create test context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds
	s.ctx = ctx
	s.ctxCancel = cancel

	s.testTopic = "/waku/2/go/filter/test"
	s.testContentTopic = "TopicA"

	s.lightNode = s.makeWakuFilterLightNode(false, true)

	s.relayNode, s.fullNode = s.makeWakuFilterFullNode(s.testTopic)

	// Re-start light node with unsupported prefix for match func
	s.lightNode.Stop()
	err := s.lightNode.CommonService.Start(s.ctx, s.lightNode.startWithIncorrectPushProto)
	s.Require().NoError(err)

	// Connect nodes
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)
	err = s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	// Subscribe
	s.contentFilter = protocol.ContentFilter{PubsubTopic: s.testTopic, ContentTopics: protocol.NewContentTopicSet(s.testContentTopic)}
	s.subDetails, err = s.lightNode.Subscribe(s.ctx, s.contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	time.Sleep(1 * time.Second)

	// Send message
	_, err = s.relayNode.PublishToTopic(s.ctx, tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch(), "second"), s.testTopic)
	s.Require().NoError(err)

	// Message should never arrive -> exit after timeout
	select {
	case msg := <-s.subDetails[0].C:
		s.log.Info("Light node received a msg")
		s.Require().Nil(msg)
	case <-time.After(1 * time.Second):
		s.Require().True(true)
	}

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)
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
	var (
		testTopic        = "/waku/2/go/filter/test"
		testContentTopic = "Topic"
		testPayload      = "test_msg"
		suffix           string
		messages         []WakuMsg
	)

	// Create test context
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 10 seconds
	s.ctx = ctx
	s.ctxCancel = cancel

	// Prepare data
	for i := 0; i < 3; i++ {
		suffix = fmt.Sprintf("%02d", i)
		messages = append(messages, WakuMsg{
			pubSubTopic:  testTopic,
			contentTopic: testContentTopic + suffix,
			payload:      testPayload,
		})
	}

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
	var (
		testTopic        = "/waku/2/go/filter/test"
		testContentTopic = "Topic"
		testPayload      = "test_msg"
		suffix           string
		messages         []WakuMsg
	)

	// Create test context
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 10 seconds
	s.ctx = ctx
	s.ctxCancel = cancel

	// Prepare data
	for i := 0; i < 2; i++ {
		suffix = fmt.Sprintf("%02d", i)
		messages = append(messages, WakuMsg{
			pubSubTopic:  testTopic + suffix,
			contentTopic: testContentTopic + suffix,
			payload:      testPayload,
		})
	}

	// Subscribe
	for _, m := range messages {
		s.subDetails = append(s.subDetails, s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())...)
		s.log.Info("Subscribing ", zap.String("PubSubTopic", m.pubSubTopic))
		_, err := s.relayNode.SubscribeToTopic(context.Background(), m.pubSubTopic)
		s.Require().NoError(err)
	}

	// Debug to see subscriptions in light node
	for _, sub := range s.subDetails {
		s.log.Info("Light Node subscription ", zap.String("PubSubTopic", sub.ContentFilter.PubsubTopic))
		s.log.Info("Light Node subscription ", zap.String("ContentTopic", sub.ContentFilter.ContentTopicsList()[0]))
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestPubSubMultiOverlapContentTopic() {
	var (
		testTopic        = "/waku/2/go/filter/test"
		testContentTopic = "Topic"
		testPayload      = "test_msg"
		suffix           string
		messages         []WakuMsg
	)

	// Create test context
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 10 seconds
	s.ctx = ctx
	s.ctxCancel = cancel

	// Prepare data
	for i := 0; i < 10; i++ {
		suffix = fmt.Sprintf("%02d", i)
		messages = append(messages, WakuMsg{
			pubSubTopic:  testTopic,
			contentTopic: testContentTopic + suffix,
			payload:      testPayload + suffix,
		})
	}

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
	var (
		testPayload = "test_msg"
		suffix      string
		messages    []WakuMsg
	)

	// Prepare data
	for i := 0; i < 2; i++ {
		suffix = fmt.Sprintf("%02d", i)
		messages = append(messages, WakuMsg{
			pubSubTopic:  s.testTopic,
			contentTopic: s.testContentTopic,
			payload:      testPayload + suffix,
		})
	}

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
	var (
		testTopic        = "/waku/2/go/filter/test"
		testContentTopic = "Topic"
		testPayload      = "test_msg"
		suffix           string
		maxContentTopics = 30
		messages         []WakuMsg
	)

	// Create test context
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 10 seconds
	s.ctx = ctx
	s.ctxCancel = cancel

	// Detect existing content topics from previous test
	if len(s.contentFilter.PubsubTopic) > 0 {
		existingTopics := len(s.contentFilter.ContentTopicsList())
		if existingTopics > 0 {
			maxContentTopics = maxContentTopics - existingTopics
		}
	}

	// Prepare data
	for i := 0; i < maxContentTopics; i++ {
		suffix = fmt.Sprintf("%02d", i)
		messages = append(messages, WakuMsg{
			pubSubTopic:  testTopic,
			contentTopic: testContentTopic + suffix,
			payload:      testPayload + suffix,
		})
	}

	// Subscribe
	for _, m := range messages {
		s.subDetails = s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	// Prepare over the limit message
	suffix = fmt.Sprintf("%02d", maxContentTopics)
	msgOverLimit := WakuMsg{
		pubSubTopic:  testTopic,
		contentTopic: testContentTopic + suffix,
		payload:      testPayload + suffix,
	}

	// Adding over the limit contentTopic should fail
	for _, sub := range s.subDetails {
		if sub.ContentFilter.PubsubTopic == msgOverLimit.pubSubTopic {
			sub.Add(msgOverLimit.contentTopic)
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds
	s.ctx = ctx
	s.ctxCancel = cancel

	fullNodeIDHex := make([]byte, hex.EncodedLen(len([]byte(s.fullNodeHost.ID()))))
	_ = hex.Encode(fullNodeIDHex, []byte(s.fullNodeHost.ID()))

	s.log.Info("Already subscribed to", zap.String("fullNode", string(fullNodeIDHex)))

	// This will overwrite values with the second node info
	s.relayNode, s.fullNode = s.makeWakuFilterFullNode(s.testTopic)

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

func (s *FilterTestSuite) BeforeTest(suiteName, testName string) {
	s.log.Info("Executing ", zap.String("testName", testName))
}

func (s *FilterTestSuite) AfterTest(suiteName, testName string) {
	s.log.Info("Finished executing ", zap.String("testName", testName))
}
