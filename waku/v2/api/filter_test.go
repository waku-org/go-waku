package api

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"go.uber.org/zap"
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

	fullNodeData2 := s.GetWakuFilterFullNode(s.TestTopic, true)
	s.ConnectHosts(s.LightNodeHost, fullNodeData2.FullNodeHost)
	peers := []peer.ID{s.FullNodeHost.ID(), fullNodeData2.FullNodeHost.ID()}
	s.Log.Info("FullNodeHost IDs:", zap.Any(peers))
	apiConfig := FilterConfig{MaxPeers: 2, Peers: peers}

	s.Require().Equal(apiConfig.MaxPeers, 2)
	s.Require().Equal(contentFilter.PubsubTopic, s.TestTopic)

	s.Log.Info("About to perform API Subscribe()")
	apiSub, err := Subscribe(context.Background(), s.LightNode, contentFilter, apiConfig)
	s.Require().NoError(err)
	s.Require().Equal(apiSub.ContentFilter, contentFilter)
	s.Log.Info("Subscribed")
}
