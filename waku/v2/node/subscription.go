package node

import (
	"sync"

	"github.com/status-im/go-waku/waku/v2/protocol"
)

// Subscription handles the subscrition to a particular pubsub topic
type Subscription struct {
	// C is channel used for receiving envelopes
	C chan *protocol.Envelope

	closed bool
	mutex  sync.Mutex
	quit   chan struct{}
}

// Unsubscribe will close a subscription from a pubsub topic. Will close the message channel
func (subs *Subscription) Unsubscribe() {
	if !subs.closed {
		close(subs.quit)
	}
}

// IsClosed determine whether a Subscription is still open for receiving messages
func (subs *Subscription) IsClosed() bool {
	subs.mutex.Lock()
	defer subs.mutex.Unlock()
	return subs.closed
}
