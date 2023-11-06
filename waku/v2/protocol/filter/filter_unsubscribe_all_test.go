package filter

func (s *FilterTestSuite) TestUnsubscribeAllWithoutContentTopics() {

	var messages = prepareData(2, false, true, true)

	// Subscribe with 2 content topics
	for _, m := range messages {
		s.subDetails = s.subscribe(m.pubSubTopic, m.contentTopic, s.fullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	// Messages should not be received for any contentTopics
	for _, m := range messages {
		s.waitForTimeout(func() {
			s.publishMsg(m.pubSubTopic, m.contentTopic, m.payload)
		}, s.subDetails[0].C)
	}
}
