package relay

import (
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"golang.org/x/exp/slices"
)

// Subscription handles the details of a particular Topic subscription. There may be many subscriptions for a given topic.
type Subscription struct {
	ID            int
	Unsubscribe   func()
	Ch            chan *protocol.Envelope
	contentFilter protocol.ContentFilter
	subType       SubsciptionType
}

type SubsciptionType int

const (
	SpecificContentTopics SubsciptionType = iota
	AllContentTopics
)

// Submit allows a message to be submitted for a subscription
func (s *Subscription) Submit(msg *protocol.Envelope) {
	//Filter and notify
	// - if contentFilter doesn't have a contentTopic
	// - if contentFilter has contentTopics and it matches with message
	if len(s.contentFilter.ContentTopicsList()) == 0 || (len(s.contentFilter.ContentTopicsList()) > 0 &&
		slices.Contains[string](s.contentFilter.ContentTopicsList(), msg.Message().ContentTopic)) {
		s.Ch <- msg
	}
}

// NewSubscription creates a subscription that will only receive messages based on the contentFilter
func NewSubscription(contentFilter protocol.ContentFilter) *Subscription {
	ch := make(chan *protocol.Envelope)
	var subType SubsciptionType
	if len(contentFilter.ContentTopicsList()) == 0 {
		subType = AllContentTopics
	}
	return &Subscription{
		Unsubscribe: func() {
			close(ch)
		}, //TODO: Need to analyze how to link this to underlying pubsub
		Ch:            ch,
		contentFilter: contentFilter,
		subType:       subType,
	}
}

// TODO: Analyze where this is used and how to address/modify this.
// ArraySubscription creates a subscription for a list of envelopes
func ArraySubscription(msgs []*protocol.Envelope) Subscription {
	ch := make(chan *protocol.Envelope, len(msgs))
	for _, msg := range msgs {
		ch <- msg
	}
	close(ch)
	return Subscription{
		Unsubscribe: func() {},
		Ch:          ch,
	}
}
