package rest

import (
	"fmt"
	"sync"

	"github.com/waku-org/go-waku/waku/v2/protocol"
	"go.uber.org/zap"
)

type MessageCache struct {
	capacity              int
	mu                    sync.RWMutex
	log                   *zap.Logger
	messages              map[string]map[string][]*RestWakuMessage
	messagesByPubsubTopic map[string][]*RestWakuMessage
}

func NewMessageCache(capacity int, log *zap.Logger) *MessageCache {
	return &MessageCache{
		capacity:              capacity,
		messages:              make(map[string]map[string][]*RestWakuMessage),
		messagesByPubsubTopic: make(map[string][]*RestWakuMessage),
		log:                   log.Named("cache"),
	}
}

func (c *MessageCache) Subscribe(contentFilter protocol.ContentFilter) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if contentFilter.PubsubTopic != "" && len(contentFilter.ContentTopics) == 0 {
		// Cache all messages that match a pubsub topic (but no content topic specified)
		// Used with named sharding
		// Eventually this must be modified once API changes to receive content topics too
		if _, ok := c.messages[contentFilter.PubsubTopic]; !ok {
			c.messagesByPubsubTopic[contentFilter.PubsubTopic] = []*RestWakuMessage{}
		}
	} else {
		// Cache messages that match a content topic, or pubsub topic + content topic
		pubSubTopicMap, err := protocol.ContentFilterToPubSubTopicMap(contentFilter)
		if err != nil {
			return err
		}

		for pubsubTopic, contentTopics := range pubSubTopicMap {
			if c.messages[pubsubTopic] == nil {
				c.messages[pubsubTopic] = make(map[string][]*RestWakuMessage)
			}

			for _, topic := range contentTopics {
				if c.messages[pubsubTopic][topic] == nil {
					c.messages[pubsubTopic][topic] = []*RestWakuMessage{}
				}
			}
		}
	}

	return nil
}

func (c *MessageCache) Unsubscribe(contentFilter protocol.ContentFilter) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if contentFilter.PubsubTopic != "" && len(contentFilter.ContentTopics) == 0 {
		// Stop caching all messages that match a pubsub topic
		// Used with named sharding
		// Eventually this must be modified once API changes to receive content topics too
		delete(c.messagesByPubsubTopic, contentFilter.PubsubTopic)
	} else {
		pubSubTopicMap, err := protocol.ContentFilterToPubSubTopicMap(contentFilter)
		if err != nil {
			return err
		}

		for pubsubTopic, contentTopics := range pubSubTopicMap {
			for _, contentTopic := range contentTopics {
				delete(c.messages[pubsubTopic], contentTopic)
			}
		}
	}

	return nil
}

func (c *MessageCache) AddMessage(envelope *protocol.Envelope) {
	c.mu.Lock()
	defer c.mu.Unlock()

	pubsubTopic := envelope.PubsubTopic()
	contentTopic := envelope.Message().ContentTopic

	message := &RestWakuMessage{}
	if err := message.FromProto(envelope.Message()); err != nil {
		c.log.Error("converting protobuffer msg into rest msg", zap.Error(err))
		return
	}

	if _, ok := c.messagesByPubsubTopic[pubsubTopic]; ok {
		c.messagesByPubsubTopic[pubsubTopic] = append(c.messagesByPubsubTopic[pubsubTopic], message)
		// Keep a specific max number of message per topic
		if len(c.messagesByPubsubTopic[pubsubTopic]) >= c.capacity {
			c.messagesByPubsubTopic[pubsubTopic] = c.messagesByPubsubTopic[pubsubTopic][1:]
		}
	}

	if c.messages[pubsubTopic] == nil || c.messages[pubsubTopic][contentTopic] == nil {
		return
	}

	// Keep a specific max number of message per topic
	if len(c.messages[pubsubTopic][contentTopic]) >= c.capacity {
		c.messages[pubsubTopic][contentTopic] = c.messages[pubsubTopic][contentTopic][1:]
	}

	c.messages[pubsubTopic][contentTopic] = append(c.messages[pubsubTopic][contentTopic], message)
}

func (c *MessageCache) GetMessages(pubsubTopic string, contentTopic string) ([]*RestWakuMessage, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.messages[pubsubTopic] == nil || c.messages[pubsubTopic][contentTopic] == nil {
		return nil, fmt.Errorf("not subscribed to pubsubTopic:%s contentTopic: %s", pubsubTopic, contentTopic)
	}
	msgs := c.messages[pubsubTopic][contentTopic]
	c.messages[pubsubTopic][contentTopic] = []*RestWakuMessage{}
	return msgs, nil
}

func (c *MessageCache) GetMessagesWithPubsubTopic(pubsubTopic string) ([]*RestWakuMessage, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.messagesByPubsubTopic[pubsubTopic] == nil {
		return nil, fmt.Errorf("not subscribed to pubsubTopic:%s", pubsubTopic)
	}

	return c.messagesByPubsubTopic[pubsubTopic], nil
}
