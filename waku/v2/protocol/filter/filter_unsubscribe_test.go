package filter

func (s *FilterTestSuite) TestUnsubscribeSingleContentTopic() {

	var newContentTopic = "TopicB"

	// Initial subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())
	s.subDetails = s.subscribe(s.testTopic, newContentTopic, s.fullNodeHost.ID())

	// Message is possible to receive for original contentTopic
	s.waitForMsg(func() {
		s.publishMsg(s.testTopic, s.testContentTopic, "test_msg")
	}, s.subDetails[0].C, false)

	// Message is possible to receive for new contentTopic
	s.waitForMsg(func() {
		s.publishMsg(s.testTopic, newContentTopic, "test_msg")
	}, s.subDetails[0].C, false)

	_ = s.unsubscribe(s.testTopic, newContentTopic, s.fullNodeHost.ID())

	// Message should not be received for new contentTopic as it was unsubscribed
	s.waitForMsg(func() {
		s.publishMsg(s.testTopic, newContentTopic, "test_msg")
	}, s.subDetails[0].C, true)

	// Message is still possible to receive for original contentTopic
	s.waitForMsg(func() {
		s.publishMsg(s.testTopic, s.testContentTopic, "test_msg2")
	}, s.subDetails[0].C, false)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestUnsubscribeMultiContentTopic() {

	messages := prepareData(3, false, true, true)

	// Subscribe with 3 content topics
	for _, m := range messages {
		s.subDetails = s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	// Unsubscribe with the last 2 content topics
	for _, m := range messages[1:] {
		_ = s.unsubscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// Messages should not be received for the last two contentTopics as it was unsubscribed
	for _, m := range messages[1:] {
		s.waitForMsg(func() {
			s.publishMsg(m.pubSubTopic, m.contentTopic, m.payload)
		}, s.subDetails[0].C, true)
	}

	// Message is still possible to receive for the first contentTopic
	s.waitForMsg(func() {
		s.publishMsg(messages[0].pubSubTopic, messages[0].contentTopic, messages[0].payload)
	}, s.subDetails[0].C, false)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}
