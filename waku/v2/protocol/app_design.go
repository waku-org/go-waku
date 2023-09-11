package protocol

import (
	"context"
	"errors"
	"sync"
)

type AppDesign struct {
	mu      *sync.RWMutex
	cancel  context.CancelFunc
	ctx     context.Context
	wg      *sync.WaitGroup
	started bool
}

func NewAppDesign() AppDesign {
	return AppDesign{
		wg: &sync.WaitGroup{},
		mu: &sync.RWMutex{},
	}
}

func (sp *AppDesign) LockAndUnlock() func() {
	sp.mu.Lock()
	return func() {
		sp.mu.Unlock()
	}
}
func (sp *AppDesign) RLockAndUnlock() func() {
	sp.mu.RLock()
	return func() {
		sp.mu.RUnlock()
	}
}

// mutex protected start function
// creates internal context over provided context and runs fn safely
// fn is excerpt to be executed to start the protocol
func (sp *AppDesign) Start(ctx context.Context, fn func() error) error {
	defer sp.LockAndUnlock()()
	if sp.started {
		return ErrAlreadyStarted
	}
	sp.started = true
	sp.ctx, sp.cancel = context.WithCancel(ctx)
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
func (sp *AppDesign) Stop(fn func()) {
	defer sp.LockAndUnlock()()
	if !sp.started {
		return
	}
	sp.cancel()
	fn()
	sp.wg.Wait()
	sp.started = false
}

func (sp AppDesign) ErrOnNotRunning() error {
	if !sp.started {
		return ErrNotStarted
	}
	return nil
}

func (sp *AppDesign) Context() context.Context {
	return sp.ctx
}
func (sp *AppDesign) WaitGroup() *sync.WaitGroup {
	return sp.wg
}
