package filter

import (
	"github.com/waku-org/go-waku/tests"
	"go.uber.org/zap"
	"strconv"
)

func (s *FilterTestSuite) TestValidPayloadsASCII() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare data
	messages := prepareData(100, false, false, true, tests.GenerateRandomASCIIString)

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsUTF8() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare data
	messages := prepareData(100, false, false, true, tests.GenerateRandomUTF8String)

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsBase64() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare data
	messages := prepareData(100, false, false, true, tests.GenerateRandomBase64String)

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsJSON() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare data
	messages := prepareData(100, false, false, true, tests.GenerateRandomJSONString)

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsURLEncoded() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare data
	messages := prepareData(100, false, false, true, tests.GenerateRandomURLEncodedString)

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsSQL() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare data
	messages := prepareData(100, false, false, true, tests.GenerateRandomSQLInsert)

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestLargePayloadsUTF8() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare basic data
	messages := prepareData(10, false, false, false, nil)

	// Generate large string
	for i := range messages {
		messages[i].payload, _ = tests.GenerateRandomUTF8String(1048576)
		s.log.Info("Generated payload with ", zap.String("length", strconv.Itoa(len(messages[i].payload))))
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}
