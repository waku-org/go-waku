package rest

import (
	v2 "github.com/status-im/go-waku/waku/v2"
	"github.com/status-im/go-waku/waku/v2/protocol"
)

type Adder func(msg *protocol.Envelope)

type runnerService struct {
	broadcaster v2.Broadcaster
	ch          chan *protocol.Envelope
	quit        chan bool
	adder       Adder
}

func newRunnerService(broadcaster v2.Broadcaster, adder Adder) *runnerService {
	return &runnerService{
		broadcaster: broadcaster,
		quit:        make(chan bool),
		adder:       adder,
	}
}

func (r *runnerService) Start() {
	r.ch = make(chan *protocol.Envelope, 1024)
	r.broadcaster.Register(nil, r.ch)
	for {
		select {
		case <-r.quit:
			return
		case envelope := <-r.ch:
			r.adder(envelope)
		}
	}
}

func (r *runnerService) Stop() {
	r.quit <- true
	r.broadcaster.Unregister(nil, r.ch)
	close(r.ch)
}
