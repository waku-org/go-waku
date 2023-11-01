package rest

import (
	"fmt"
	"sync"

	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

type filterCache struct {
	capacity int
	mu       sync.RWMutex
	data     map[string]map[string][]*pb.WakuMessage
}

func newFilterCache(capacity int) *filterCache {
	return &filterCache{
		capacity: capacity,
		data:     make(map[string]map[string][]*pb.WakuMessage),
	}
}

func (c *filterCache) subscribe(contentFilter protocol.ContentFilter) {
	c.mu.Lock()
	defer c.mu.Unlock()

	pubsubTopic := contentFilter.PubsubTopic
	if c.data[pubsubTopic] == nil {
		c.data[pubsubTopic] = make(map[string][]*pb.WakuMessage)
	}
	for topic := range contentFilter.ContentTopics {
		c.data[pubsubTopic][topic] = []*pb.WakuMessage{}
	}
}

func (c *filterCache) unsubscribe(pubsubTopic string, contentTopic string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data[pubsubTopic], contentTopic)
}

func (c *filterCache) addMessage(envelope *protocol.Envelope) {
	c.mu.Lock()
	defer c.mu.Unlock()

	pubsubTopic := envelope.PubsubTopic()
	contentTopic := envelope.Message().ContentTopic
	if c.data[pubsubTopic] == nil || c.data[pubsubTopic][contentTopic] == nil {
		return
	}

	// Keep a specific max number of message per topic
	if len(c.data[pubsubTopic][contentTopic]) >= c.capacity {
		c.data[pubsubTopic][contentTopic] = c.data[pubsubTopic][contentTopic][1:]
	}

	c.data[pubsubTopic][contentTopic] = append(c.data[pubsubTopic][contentTopic], envelope.Message())
}

func (c *filterCache) getMessages(pubsubTopic string, contentTopic string) ([]*pb.WakuMessage, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data[pubsubTopic] == nil || c.data[pubsubTopic][contentTopic] == nil {
		return nil, fmt.Errorf("Not subscribed to pubsubTopic:%s contentTopic: %s", pubsubTopic, contentTopic)
	}
	msgs := c.data[pubsubTopic][contentTopic]
	c.data[pubsubTopic][contentTopic] = nil
	return msgs, nil
}
func (c *filterCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = map[string]map[string][]*pb.WakuMessage{}
}
