package rpc

import (
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

type Adder func(msg *protocol.Envelope)

type runnerService struct {
	broadcaster relay.Broadcaster
	sub         relay.Subscription
	adder       Adder
}

func newRunnerService(broadcaster relay.Broadcaster, adder Adder) *runnerService {
	return &runnerService{
		broadcaster: broadcaster,
		adder:       adder,
	}
}

func (r *runnerService) Start() {
	r.sub = r.broadcaster.RegisterForAll(1024)
	for envelope := range r.sub.Ch {
		r.adder(envelope)
	}
}

func (r *runnerService) Stop() {
	r.sub.Unsubscribe()
}
