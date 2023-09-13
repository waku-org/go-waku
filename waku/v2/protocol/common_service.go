package protocol

import (
	"context"
	"errors"
	"sync"
)

type CommonService struct {
	sync.RWMutex
	cancel  context.CancelFunc
	ctx     context.Context
	wg      sync.WaitGroup
	started bool
}

func NewCommonService() *CommonService {
	return &CommonService{
		wg:      sync.WaitGroup{},
		RWMutex: sync.RWMutex{},
	}
}

// mutex protected start function
// creates internal context over provided context and runs fn safely
// fn is excerpt to be executed to start the protocol
func (sp *CommonService) Start(ctx context.Context, fn func() error) error {
	sp.Lock()
	defer sp.Unlock()
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
func (sp *CommonService) Stop(fn func()) {
	sp.Lock()
	defer sp.Unlock()
	if !sp.started {
		return
	}
	sp.cancel()
	fn()
	sp.wg.Wait()
	sp.started = false
}

func (sp *CommonService) ErrOnNotRunning() error {
	if !sp.started {
		return ErrNotStarted
	}
	return nil
}

func (sp *CommonService) Context() context.Context {
	return sp.ctx
}
func (sp *CommonService) WaitGroup() *sync.WaitGroup {
	return &sp.wg
}
