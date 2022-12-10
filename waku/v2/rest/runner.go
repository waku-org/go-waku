package rest

import (
	"context"

	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

type Adder func(msg *protocol.Envelope)

type runnerService struct {
	broadcaster v2.Broadcaster
	ch          chan *protocol.Envelope
	cancel      context.CancelFunc
	adder       Adder
}

func newRunnerService(broadcaster v2.Broadcaster, adder Adder) *runnerService {
	return &runnerService{
		broadcaster: broadcaster,
		adder:       adder,
	}
}

func (r *runnerService) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	r.ch = make(chan *protocol.Envelope, 1024)
	r.cancel = cancel
	r.broadcaster.Register(nil, r.ch)
	for {
		select {
		case <-ctx.Done():
			return
		case envelope := <-r.ch:
			r.adder(envelope)
		}
	}
}

func (r *runnerService) Stop() {
	r.cancel()
	r.broadcaster.Unregister(nil, r.ch)
	close(r.ch)
}
