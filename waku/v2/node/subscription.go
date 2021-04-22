package node

import (
	"sync"

	"github.com/status-im/go-waku/waku/v2/protocol"
)

type Subscription struct {
	C      chan *protocol.Envelope
	closed bool
	mutex  sync.Mutex
	quit   chan struct{}
}

func (subs *Subscription) Unsubscribe() {
	if !subs.closed {
		close(subs.quit)
	}
}

func (subs *Subscription) IsClosed() bool {
	subs.mutex.Lock()
	defer subs.mutex.Unlock()
	return subs.closed
}
