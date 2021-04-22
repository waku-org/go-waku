package node

import (
	"github.com/status-im/go-waku/waku/v2/protocol"
)

// Adapted from https://github.com/dustin/go-broadcast/commit/f664265f5a662fb4d1df7f3533b1e8d0e0277120 which was released under MIT license

type broadcaster struct {
	input chan *protocol.Envelope
	reg   chan chan<- *protocol.Envelope
	unreg chan chan<- *protocol.Envelope

	outputs map[chan<- *protocol.Envelope]bool
}

// The Broadcaster interface describes the main entry points to
// broadcasters.
type Broadcaster interface {
	// Register a new channel to receive broadcasts
	Register(chan<- *protocol.Envelope)
	// Unregister a channel so that it no longer receives broadcasts.
	Unregister(chan<- *protocol.Envelope)
	// Shut this broadcaster down.
	Close() error
	// Submit a new object to all subscribers
	Submit(*protocol.Envelope)
}

func (b *broadcaster) broadcast(m *protocol.Envelope) {
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
		input:   make(chan *protocol.Envelope, buflen),
		reg:     make(chan chan<- *protocol.Envelope),
		unreg:   make(chan chan<- *protocol.Envelope),
		outputs: make(map[chan<- *protocol.Envelope]bool),
	}

	go b.run()

	return b
}

func (b *broadcaster) Register(newch chan<- *protocol.Envelope) {
	b.reg <- newch
}

func (b *broadcaster) Unregister(newch chan<- *protocol.Envelope) {
	b.unreg <- newch
}

func (b *broadcaster) Close() error {
	close(b.reg)
	return nil
}

func (b *broadcaster) Submit(m *protocol.Envelope) {
	if b != nil {
		b.input <- m
	}
}
