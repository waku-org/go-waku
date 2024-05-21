package api

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

func TestFilterApiSuite(t *testing.T) {
	suite.Run(t, new(FilterApiTestSuite))
}

type FilterApiTestSuite struct {
	filter.FilterTestSuite
}

func (s *FilterApiTestSuite) SetupTest() {
	s.FilterTestSuite.SetupTest()
	s.Log.Info("SetupTest()")
}

func (s *FilterApiTestSuite) TearDownTest() {
	s.FilterTestSuite.TearDownTest()
}

func (s *FilterApiTestSuite) TestSubscribe() {
	contentFilter := protocol.ContentFilter{PubsubTopic: s.TestTopic, ContentTopics: protocol.NewContentTopicSet(s.TestContentTopic)}

	// We have one full node already created in SetupTest(),
	// create another one
	fullNodeData2 := s.GetWakuFilterFullNode(s.TestTopic, true)
	s.ConnectToFullNode(s.LightNode, fullNodeData2.FullNode)
	//s.ConnectHosts(s.FullNodeHost, fullNodeData2.FullNodeHost)
	peers := []peer.ID{s.FullNodeHost.ID(), fullNodeData2.FullNodeHost.ID()}
	s.Log.Info("FullNodeHost IDs:", zap.Any("peers", peers))
	// Make sure IDs are different
	//s.Require().True(peers[0] != peers[1])
	apiConfig := FilterConfig{MaxPeers: 2}

	s.Require().Equal(apiConfig.MaxPeers, 2)
	s.Require().Equal(contentFilter.PubsubTopic, s.TestTopic)

	s.Log.Info("About to perform API Subscribe()")
	apiSub, err := Subscribe(context.Background(), s.LightNode, contentFilter, apiConfig, s.Log)
	s.Require().NoError(err)
	s.Require().Equal(apiSub.ContentFilter, contentFilter)
	s.Log.Info("Subscribed")

	s.Require().Len(apiSub.subs, 2)
	for sub := range apiSub.subs {
		s.Log.Info("SubDetails:", zap.String("id", sub))
	}
	subsArray := maps.Keys(apiSub.subs)
	s.Require().True(subsArray[0] != subsArray[1])
	// Publish msg and confirm it's received twice because of multiplexing
	s.PublishMsg(&filter.WakuMsg{PubSubTopic: s.TestTopic, ContentTopic: s.TestContentTopic, Payload: "Test msg"})
	cnt := 0
	for msg := range apiSub.DataCh {
		s.Log.Info("Received msg:", zap.Int("cnt", cnt), zap.String("payload", string(msg.Message().Payload)))
		cnt++
		break
	}
	s.Require().Equal(cnt, 1)

	//Verify HealthCheck
	subs := s.LightNode.Subscriptions()
	s.Require().Equal(2, len(subs))

	s.Log.Info("stopping full node", zap.Stringer("id", fullNodeData2.FullNodeHost.ID()))
	fullNodeData3 := s.GetWakuFilterFullNode(s.TestTopic, true)

	s.ConnectToFullNode(s.LightNode, fullNodeData3.FullNode)

	fullNodeData2.FullNode.Stop()
	fullNodeData2.FullNodeHost.Close()
	time.Sleep(2 * time.Second)
	s.Require().Equal(2, len(apiSub.subs))

	for subId := range apiSub.subs {
		sub := apiSub.subs[subId]
		s.Require().NotEqual(fullNodeData2.FullNodeHost.ID(), sub.PeerID)
	}

	apiSub.Unsubscribe()
	for range apiSub.DataCh {
	}
	s.Log.Info("DataCh is closed")

}
