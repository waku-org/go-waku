package filter

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/stretchr/testify/suite"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/service"
	"go.uber.org/zap"
)

func TestFilterSuite(t *testing.T) {
	suite.Run(t, new(FilterTestSuite))
}

func (s *FilterTestSuite) TestRunningGuard() {

	s.LightNode.Stop()

	contentFilter := protocol.ContentFilter{PubsubTopic: "test", ContentTopics: protocol.NewContentTopicSet("test")}

	_, err := s.LightNode.Subscribe(s.ctx, contentFilter, WithPeer(s.FullNodeHost.ID()))

	s.Require().ErrorIs(err, service.ErrNotStarted)

	err = s.LightNode.Start(s.ctx)
	s.Require().NoError(err)

	_, err = s.LightNode.Subscribe(s.ctx, contentFilter, WithPeer(s.FullNodeHost.ID()))

	s.Require().NoError(err)
}

func (s *FilterTestSuite) TestFireAndForgetAndCustomWg() {

	contentFilter := protocol.ContentFilter{PubsubTopic: "test", ContentTopics: protocol.NewContentTopicSet("test")}

	_, err := s.LightNode.Subscribe(s.ctx, contentFilter, WithPeer(s.FullNodeHost.ID()))
	s.Require().NoError(err)

	result, err := s.LightNode.Unsubscribe(s.ctx, contentFilter, DontWait())
	s.Require().NoError(err)
	s.Require().Equal(0, len(result.Errors()))

	_, err = s.LightNode.Subscribe(s.ctx, contentFilter, WithPeer(s.FullNodeHost.ID()))
	s.Require().NoError(err)

	wg := sync.WaitGroup{}
	_, err = s.LightNode.Unsubscribe(s.ctx, contentFilter, WithWaitGroup(&wg))
	wg.Wait()
	s.Require().NoError(err)
}

func (s *FilterTestSuite) TestStartStop() {

	var wg sync.WaitGroup
	wg.Add(2)
	s.MakeWakuFilterLightNode()

	stopNode := func() {
		for i := 0; i < 100000; i++ {
			s.LightNode.Stop()
		}
		wg.Done()
	}

	startNode := func() {
		for i := 0; i < 100; i++ {
			err := s.LightNode.Start(context.Background())
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
	s.LightNode.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 10 seconds
	s.ctx = ctx
	s.ctxCancel = cancel

	cTopic1Str := "0/test/1/testTopic/proto"
	cTopic1, err := protocol.StringToContentTopic(cTopic1Str)
	s.Require().NoError(err)
	//Computing pubSubTopic only for filterFullNode.
	pubSubTopic := protocol.GetShardFromContentTopic(cTopic1, protocol.GenerationZeroShardsCount)
	s.TestContentTopic = cTopic1Str
	s.TestTopic = pubSubTopic.String()

	s.MakeWakuFilterLightNode()
	s.StartLightNode()
	s.MakeWakuFilterFullNode(pubSubTopic.String(), false)

	s.LightNodeHost.Peerstore().AddAddr(s.FullNodeHost.ID(), tests.GetHostAddress(s.FullNodeHost), peerstore.PermanentAddrTTL)
	err = s.LightNodeHost.Peerstore().AddProtocols(s.FullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	s.Log.Info("Testing Autoshard:CreateSubscription")
	s.subscribe("", s.TestContentTopic, s.FullNodeHost.ID())
	s.waitForMsg(&WakuMsg{s.TestTopic, s.TestContentTopic, ""})

	// Wrong content topic
	s.waitForTimeout(&WakuMsg{s.TestTopic, "TopicB", "second"})

	_, err = s.LightNode.Unsubscribe(s.ctx, s.contentFilter, WithPeer(s.FullNodeHost.ID()))
	s.Require().NoError(err)

	time.Sleep(1 * time.Second)

	// Should not receive after unsubscribe
	s.waitForTimeout(&WakuMsg{s.TestTopic, s.TestContentTopic, "third"})

	s.subscribe("", s.TestContentTopic, s.FullNodeHost.ID())

	s.Log.Info("Testing Autoshard:SubscriptionPing")
	err = s.LightNode.Ping(context.Background(), s.FullNodeHost.ID())
	s.Require().NoError(err)

	// Test ModifySubscription Subscribe to another content_topic
	s.Log.Info("Testing Autoshard:ModifySubscription")

	newContentTopic := "0/test/1/testTopic1/proto"
	s.subscribe("", newContentTopic, s.FullNodeHost.ID())

	s.waitForMsg(&WakuMsg{s.TestTopic, newContentTopic, ""})

	_, err = s.LightNode.Unsubscribe(s.ctx, protocol.ContentFilter{
		PubsubTopic:   s.TestTopic,
		ContentTopics: protocol.NewContentTopicSet(newContentTopic),
	})
	s.Require().NoError(err)

	_, err = s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestLightNodeIsListening() {

	messages := s.prepareData(2, true, true, false, nil)

	// Subscribe with the first message only
	s.subscribe(messages[0].PubSubTopic, messages[0].ContentTopic, s.FullNodeHost.ID())

	// IsListening returns true for the first message
	listenStatus := s.LightNode.IsListening(messages[0].PubSubTopic, messages[0].ContentTopic)
	s.Require().True(listenStatus)

	// IsListening returns false for the second message
	listenStatus = s.LightNode.IsListening(messages[1].PubSubTopic, messages[1].ContentTopic)
	s.Require().False(listenStatus)

	// IsListening returns false for combination as well
	listenStatus = s.LightNode.IsListening(messages[0].PubSubTopic, messages[1].ContentTopic)
	s.Require().False(listenStatus)

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)
}

func (s *FilterTestSuite) BeforeTest(suiteName, testName string) {
	s.Log.Info("Executing ", zap.String("testName", testName))
}

func (s *FilterTestSuite) AfterTest(suiteName, testName string) {
	s.Log.Info("Finished executing ", zap.String("testName", testName))
}
