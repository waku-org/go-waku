package filter

import (
	"context"
	"net/http"
)

func (s *FilterTestSuite) TestSubscriptionPing() {
	err := s.lightNode.Ping(context.Background(), s.fullNodeHost.ID())
	s.Require().Error(err)
	filterErr, ok := err.(*FilterError)
	s.Require().True(ok)
	s.Require().Equal(filterErr.Code, http.StatusNotFound)

	contentTopic := "abc"
	s.subscribe(s.testTopic, contentTopic, s.fullNodeHost.ID())

	err = s.lightNode.Ping(context.Background(), s.fullNodeHost.ID())
	s.Require().NoError(err)
}

func (s *FilterTestSuite) TestUnSubscriptionPing() {

	s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	err := s.lightNode.Ping(context.Background(), s.fullNodeHost.ID())
	s.Require().NoError(err)

	_, err = s.lightNode.Unsubscribe(s.ctx, s.contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	err = s.lightNode.Ping(context.Background(), s.fullNodeHost.ID())
	s.Require().Error(err)
}
