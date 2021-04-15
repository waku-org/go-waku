package node

import (
	"github.com/status-im/go-waku/waku/common"
)

// Adapted from https://github.com/dustin/go-broadcast/commit/f664265f5a662fb4d1df7f3533b1e8d0e0277120 which was released under MIT license

type broadcaster struct {
	input chan *common.Envelope
	reg   chan chan<- *common.Envelope
	unreg chan chan<- *common.Envelope

	outputs map[chan<- *common.Envelope]bool
}

// The Broadcaster interface describes the main entry points to
// broadcasters.
type Broadcaster interface {
	// Register a new channel to receive broadcasts
	Register(chan<- *common.Envelope)
	// Unregister a channel so that it no longer receives broadcasts.
	Unregister(chan<- *common.Envelope)
	// Shut this broadcaster down.
	Close() error
	// Submit a new object to all subscribers
	Submit(*common.Envelope)
}

func (b *broadcaster) broadcast(m *common.Envelope) {
	for ch := range b.outputs {
		ch <- m
	}
}

func (b *broadcaster) run() {
	for {
		select {
		case m := <-b.input:
			b.broadcast(m)
		case ch, ok := <-b.reg:
			if ok {
				b.outputs[ch] = true
			} else {
				return
			}
		case ch := <-b.unreg:
			delete(b.outputs, ch)
		}
	}
}

func NewBroadcaster(buflen int) Broadcaster {
	b := &broadcaster{
		input:   make(chan *common.Envelope, buflen),
		reg:     make(chan chan<- *common.Envelope),
		unreg:   make(chan chan<- *common.Envelope),
		outputs: make(map[chan<- *common.Envelope]bool),
	}

	go b.run()

	return b
}

func (b *broadcaster) Register(newch chan<- *common.Envelope) {
	b.reg <- newch
}

func (b *broadcaster) Unregister(newch chan<- *common.Envelope) {
	b.unreg <- newch
}

func (b *broadcaster) Close() error {
	close(b.reg)
	return nil
}

func (b *broadcaster) Submit(m *common.Envelope) {
	if b != nil {
		b.input <- m
	}
}
