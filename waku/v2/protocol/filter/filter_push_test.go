package filter

import (
	"context"
	"github.com/waku-org/go-waku/tests"
	"time"
)

func (s *FilterTestSuite) TestValidPayloadsASCII() {

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second)

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare basic data
	messages := prepareData(100, false, false, true, tests.GenerateRandomASCIIString)

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsUTF8() {

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second)

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare basic data
	messages := prepareData(100, false, false, true, tests.GenerateRandomUTF8String)

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}
