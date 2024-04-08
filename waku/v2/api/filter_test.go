package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
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
	apiConfig := FilterConfig{MaxPeers: 2}
	apiSub, err := Subscribe(context.Background(), s.LightNode, contentFilter, apiConfig)
	s.Require().NoError(err)

	s.Require().NotNil(apiSub.wf)
}
