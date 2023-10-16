package relay

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/waku-org/go-waku/waku/v2/protocol"
)

type subscriptions struct {
	mu           sync.RWMutex
	topicsToSubs map[string]map[int]*Subscription //map of pubSubTopic to subscriptions
	id           int
}

func newSubStore() subscriptions {
	return subscriptions{
		topicsToSubs: make(map[string]map[int]*Subscription),
	}
}
func (s *subscriptions) createNewSubscription(contentFilter protocol.ContentFilter, chLen int) Subscription {
	ch := make(chan *protocol.Envelope, chLen)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.id++
	pubsubTopic := contentFilter.PubsubTopic
	if s.topicsToSubs[pubsubTopic] == nil {
		s.topicsToSubs[pubsubTopic] = make(map[int]*Subscription)
	}
	id := s.id
	sub := Subscription{
		ID: id,
		// read only channel,will not block forever, returns once closed.
		Ch: ch,
		// Unsubscribe function is safe, can be called multiple times
		// and even after broadcaster has stopped running.
		Unsubscribe: func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.topicsToSubs[pubsubTopic] == nil {
				return
			}
			if sub := s.topicsToSubs[pubsubTopic][id]; sub != nil {
				close(sub.Ch)
				delete(s.topicsToSubs[pubsubTopic], id)
			}
		},
		contentFilter: contentFilter,
	}
	s.topicsToSubs[pubsubTopic][id] = &sub
	return sub
}

func (s *subscriptions) broadcast(ctx context.Context, m *protocol.Envelope) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sub := range s.topicsToSubs[m.PubsubTopic()] {
		select {
		// using ctx.Done for returning on cancellation is needed
		// reason:
		// if for a channel there is no one listening to it
		// the broadcast will acquire lock and wait until there is a receiver on that channel.
		// this will also block the chStore close function as it uses same mutex
		case <-ctx.Done():
			return
		default:
			sub.Submit(m)
		}
	}

	// send to all wildcard subscribers
	for _, sub := range s.topicsToSubs[""] {
		select {
		case <-ctx.Done():
			return
		case sub.Ch <- m:
		}
	}
}

func (s *subscriptions) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, subs := range s.topicsToSubs {
		for _, sub := range subs {
			close(sub.Ch)
		}
	}
	s.topicsToSubs = nil
}

// Broadcaster is used to create a fanout for an envelope that will be received by any subscriber interested in the topic of the message
type Broadcaster interface {
	Start(ctx context.Context) error
	Stop()
	Register(contentFilter protocol.ContentFilter, chLen ...int) Subscription
	RegisterForAll(chLen ...int) Subscription
	Submit(*protocol.Envelope)
}

// ////
// thread safe
// panic safe, input can't be submitted to `input` channel after stop
// lock safe, only read channels are returned and later closed, calling code has guarantee Register channel will not block forever.
// no opened channel leaked, all created only read channels are closed when stop
// even if there is noone listening to returned channels, guarantees to be lockfree.
type broadcaster struct {
	bufLen int
	cancel context.CancelFunc
	input  chan *protocol.Envelope
	//
	chStore subscriptions
	running atomic.Bool
}

// NewBroadcaster creates a new instance of a broadcaster
func NewBroadcaster(bufLen int) *broadcaster {
	return &broadcaster{
		bufLen: bufLen,
	}
}

// Start initiates the execution of the broadcaster
func (b *broadcaster) Start(ctx context.Context) error {
	if !b.running.CompareAndSwap(false, true) { // if not running then start
		return errors.New("already started")
	}
	ctx, cancel := context.WithCancel(ctx)
	b.cancel = cancel
	b.chStore = newSubStore()
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
				b.chStore.broadcast(ctx, msg)
			}
		}
	}
}

// Stop stops the execution of the broadcaster and closes all subscriptions
func (b *broadcaster) Stop() {
	if !b.running.CompareAndSwap(true, false) { // if running then stop
		return
	}
	// cancel must be before chStore.close(), so that broadcast releases lock before chStore.close() acquires it.
	b.cancel()        // exit the run loop,
	b.chStore.close() // close all channels that we send to
	close(b.input)    // close input channel
}

// Register returns a subscription for an specific pubsub topic and/or list of contentTopics
func (b *broadcaster) Register(contentFilter protocol.ContentFilter, chLen ...int) Subscription {
	return b.chStore.createNewSubscription(contentFilter, getChLen(chLen))
}

// RegisterForAll returns a subscription for all topics
func (b *broadcaster) RegisterForAll(chLen ...int) Subscription {
	return b.chStore.createNewSubscription(protocol.NewContentFilter(""), getChLen(chLen))
}

func getChLen(chLen []int) int {
	l := 0
	if len(chLen) > 0 {
		l = chLen[0]
	}
	return l
}

// Submit is used to broadcast messages to subscribers. It only accepts value when running.
func (b *broadcaster) Submit(m *protocol.Envelope) {
	if b.running.Load() {
		b.input <- m
	}
}
