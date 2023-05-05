package relay

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/waku-org/go-waku/waku/v2/protocol"
)

type chStore struct {
	mu           sync.RWMutex
	topicToChans map[string]map[int]chan *protocol.Envelope
	id           int
}

func newChStore() chStore {
	return chStore{
		topicToChans: make(map[string]map[int]chan *protocol.Envelope),
	}
}
func (s *chStore) getNewCh(topic string, chLen int) Subscription {
	ch := make(chan *protocol.Envelope, chLen)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.id++
	//
	if s.topicToChans[topic] == nil {
		s.topicToChans[topic] = make(map[int]chan *protocol.Envelope)
	}
	id := s.id
	s.topicToChans[topic][id] = ch
	return Subscription{
		// read only channel , will not block forever, return once closed.
		Ch: ch,
		// Unsubscribe function is safe, can be called multiple times
		// and even after broadcaster has stopped running.
		Unsubscribe: func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.topicToChans[topic] == nil {
				return
			}
			if ch := s.topicToChans[topic][id]; ch != nil {
				close(ch)
				delete(s.topicToChans[topic], id)
			}
		},
	}
}

func (s *chStore) broadcast(m *protocol.Envelope) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fmt.Println(m)
	fmt.Println(s.topicToChans, "msg")
	for _, ch := range s.topicToChans[m.PubsubTopic()] {
		fmt.Println(m.PubsubTopic())
		ch <- m
	}
	// send to all registered subscribers
	for _, ch := range s.topicToChans[""] {
		ch <- m
	}
}

func (b *chStore) close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, chans := range b.topicToChans {
		for _, ch := range chans {
			close(ch)
		}
	}
	b.topicToChans = nil
}

type Broadcaster interface {
	Start(ctx context.Context) error
	Stop()
	Register(topic string, chLen ...int) Subscription
	RegisterForAll(chLen ...int) Subscription
	Submit(*protocol.Envelope)
}

// ////
// thread safe
// panic safe, input can't be submitted to `input` channel after stop
// lock safe, only read channels are returned and later closed, calling code has guarantee Register channel will not block forever.
// no opened channel leaked, all created only read channels are closed when stop
type broadcaster struct {
	bufLen int
	cancel context.CancelFunc
	input  chan *protocol.Envelope
	//
	chStore chStore
	running atomic.Bool
}

func NewBroadcaster(bufLen int) *broadcaster {
	return &broadcaster{
		bufLen: bufLen,
	}
}

func (b *broadcaster) Start(ctx context.Context) error {
	if !b.running.CompareAndSwap(false, true) { // if not running then start
		return errors.New("already started")
	}
	ctx, cancel := context.WithCancel(ctx)
	b.cancel = cancel
	b.chStore = newChStore()
	b.input = make(chan *protocol.Envelope, b.bufLen)
	go b.run(ctx)
	return nil
}

func (b *broadcaster) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-b.input:
			if ok {
				b.chStore.broadcast(msg)
			}
		}
	}
}

func (b *broadcaster) Stop() {
	if !b.running.CompareAndSwap(true, false) { // if running then stop
		return
	}
	b.chStore.close() // close all channels that we send to
	close(b.input)    // close input channel
	b.cancel()        // exit the run loop
}

// returned subscription is all speicfied topic
func (b *broadcaster) Register(topic string, chLen ...int) Subscription {
	return b.chStore.getNewCh(topic, getChLen(chLen))
}

// return subscription is for all topic
func (b *broadcaster) RegisterForAll(chLen ...int) Subscription {

	return b.chStore.getNewCh("", getChLen(chLen))
}

func getChLen(chLen []int) int {
	l := 0
	if len(chLen) > 0 {
		l = chLen[0]
	}
	return l
}

// only accepts value when running.
func (b *broadcaster) Submit(m *protocol.Envelope) {
	if b.running.Load() {
		b.input <- m
	}
}
