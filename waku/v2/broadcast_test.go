package v2

import (
	"sync"
	"testing"

	"github.com/status-im/go-waku/waku/v2/protocol"
)

// Adapted from https://github.com/dustin/go-broadcast/commit/f664265f5a662fb4d1df7f3533b1e8d0e0277120
// by Dustin Sallings (c) 2013, which was released under MIT license
func TestBroadcast(t *testing.T) {
	wg := sync.WaitGroup{}

	b := NewBroadcaster(100)
	defer b.Close()

	for i := 0; i < 5; i++ {
		wg.Add(1)

		cch := make(chan *protocol.Envelope)
		b.Register(nil, cch)
		go func() {
			defer wg.Done()
			defer b.Unregister(nil, cch)
			<-cch
		}()

	}

	env := new(protocol.Envelope)
	b.Submit(env)

	wg.Wait()
}

func TestBroadcastWait(t *testing.T) {
	wg := sync.WaitGroup{}

	b := NewBroadcaster(100)
	defer b.Close()

	for i := 0; i < 5; i++ {
		wg.Add(1)

		cch := make(chan *protocol.Envelope)
		<-b.WaitRegister(nil, cch)

		go func() {
			defer wg.Done()

			<-cch
			<-b.WaitUnregister(nil, cch)
		}()

	}

	env := new(protocol.Envelope)
	b.Submit(env)

	wg.Wait()
}

func TestBroadcastCleanup(t *testing.T) {
	b := NewBroadcaster(100)
	topic := "test"
	b.Register(&topic, make(chan *protocol.Envelope))
	b.Close()
}
