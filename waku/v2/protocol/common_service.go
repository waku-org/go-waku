package protocol

import (
	"context"
	"errors"
	"sync"
)

// this is common layout for all the services that require mutex protection and a guarantee that all running goroutines will be finished before stop finishes execution. This guarantee comes from waitGroup all one has to use CommonService.WaitGroup() in the goroutines that should finish by the end of stop function.
type CommonService[T any] struct {
	sync.RWMutex
	cancel  context.CancelFunc
	ctx     context.Context
	wg      sync.WaitGroup
	started bool
	channel chan T
}

func NewCommonService[T any]() *CommonService[T] {
	return &CommonService[T]{
		wg:      sync.WaitGroup{},
		RWMutex: sync.RWMutex{},
	}
}

// mutex protected start function
// creates internal context over provided context and runs fn safely
// fn is excerpt to be executed to start the protocol
func (sp *CommonService[T]) Start(ctx context.Context, fn func() error) error {
	sp.Lock()
	defer sp.Unlock()
	if sp.started {
		return ErrAlreadyStarted
	}
	sp.started = true
	sp.ctx, sp.cancel = context.WithCancel(ctx)
	// currently is used in discv5 for returning new discovered Peers to peerConnector for connecting with them
	sp.channel = make(chan T)
	if err := fn(); err != nil {
		sp.started = false
		sp.cancel()
		return err
	}
	return nil
}

var ErrAlreadyStarted = errors.New("already started")
var ErrNotStarted = errors.New("not started")

// mutex protected stop function
func (sp *CommonService[T]) Stop(fn func()) {
	sp.Lock()
	defer sp.Unlock()
	if !sp.started {
		return
	}
	sp.cancel()
	fn()
	sp.wg.Wait()
	close(sp.channel)
	sp.started = false
}

// This is not a mutex protected function, it is up to the caller to use it in a mutex protected context
func (sp *CommonService[T]) ErrOnNotRunning() error {
	if !sp.started {
		return ErrNotStarted
	}
	return nil
}

func (sp *CommonService[T]) Context() context.Context {
	return sp.ctx
}
func (sp *CommonService[T]) WaitGroup() *sync.WaitGroup {
	return &sp.wg
}
func (sp *CommonService[T]) GetListeningChan() <-chan T {
	return sp.channel
}
func (sp *CommonService[T]) PushToChan(data T) bool {
	sp.RLock()
	defer sp.RUnlock()
	if !sp.started {
		return false
	}
	select {
	case sp.channel <- data:
		return true
	case <-sp.ctx.Done():
		return false
	}
}
