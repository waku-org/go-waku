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
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

func TestFilterSuite(t *testing.T) {
	suite.Run(t, new(FilterTestSuite))
}

const defaultTestPubSubTopic = "/waku/2/go/filter/test"
const defaultTestContentTopic = "/test/10/my-app"

func (s *FilterTestSuite) SetupTest() {
	log := utils.Logger() //.Named("filterv2-test")
	s.Log = log
	// Use a pointer to WaitGroup so that to avoid copying
	// https://pkg.go.dev/sync#WaitGroup
	s.wg = &sync.WaitGroup{}

	// Create test context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds
	s.ctx = ctx
	s.ctxCancel = cancel

	s.testTopic = defaultTestPubSubTopic
	s.testContentTopic = defaultTestContentTopic

	s.MakeWakuFilterLightNode()
	s.StartLightNode()

	//TODO: Add tests to verify broadcaster.

	s.MakeWakuFilterFullNode(s.testTopic, false)

	// Connect nodes
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)
	err := s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TearDownTest() {
	s.fullNode.Stop()
	s.relayNode.Stop()
	s.RelaySub.Unsubscribe()
	s.lightNode.Stop()
	s.ctxCancel()
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
	s.MakeWakuFilterLightNode()

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

	s.MakeWakuFilterLightNode()
	s.StartLightNode()
	s.MakeWakuFilterFullNode(pubSubTopic.String(), false)

	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)
	err = s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	s.Log.Info("Testing Autoshard:CreateSubscription")
	s.subDetails = s.subscribe("", s.testContentTopic, s.fullNodeHost.ID())
	s.waitForMsg(&WakuMsg{s.testTopic, s.testContentTopic, ""})

	// Wrong content topic
	s.waitForTimeout(&WakuMsg{s.testTopic, "TopicB", "second"})

	_, err = s.lightNode.Unsubscribe(s.ctx, s.contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	time.Sleep(1 * time.Second)

	// Should not receive after unsubscribe
	s.waitForTimeout(&WakuMsg{s.testTopic, s.testContentTopic, "third"})

	s.subDetails = s.subscribe("", s.testContentTopic, s.fullNodeHost.ID())

	s.Log.Info("Testing Autoshard:SubscriptionPing")
	err = s.lightNode.Ping(context.Background(), s.fullNodeHost.ID())
	s.Require().NoError(err)

	// Test ModifySubscription Subscribe to another content_topic
	s.Log.Info("Testing Autoshard:ModifySubscription")

	newContentTopic := "0/test/1/testTopic1/proto"
	s.subDetails = s.subscribe("", newContentTopic, s.fullNodeHost.ID())

	s.waitForMsg(&WakuMsg{s.testTopic, newContentTopic, ""})

	_, err = s.lightNode.Unsubscribe(s.ctx, protocol.ContentFilter{
		PubsubTopic:   s.testTopic,
		ContentTopics: protocol.NewContentTopicSet(newContentTopic),
	})
	s.Require().NoError(err)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestLightNodeIsListening() {

	messages := prepareData(2, true, true, false, nil)

	// Subscribe with the first message only
	s.subDetails = s.subscribe(messages[0].pubSubTopic, messages[0].contentTopic, s.fullNodeHost.ID())

	// IsListening returns true for the first message
	listenStatus := s.lightNode.IsListening(messages[0].pubSubTopic, messages[0].contentTopic)
	s.Require().True(listenStatus)

	// IsListening returns false for the second message
	listenStatus = s.lightNode.IsListening(messages[1].pubSubTopic, messages[1].contentTopic)
	s.Require().False(listenStatus)

	// IsListening returns false for combination as well
	listenStatus = s.lightNode.IsListening(messages[0].pubSubTopic, messages[1].contentTopic)
	s.Require().False(listenStatus)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)
}

func (s *FilterTestSuite) BeforeTest(suiteName, testName string) {
	s.Log.Info("Executing ", zap.String("testName", testName))
}

func (s *FilterTestSuite) AfterTest(suiteName, testName string) {
	s.Log.Info("Finished executing ", zap.String("testName", testName))
}
