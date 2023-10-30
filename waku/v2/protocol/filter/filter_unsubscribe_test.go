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
