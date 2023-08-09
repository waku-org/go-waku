package gowaku

import (
	"github.com/waku-org/go-waku/library"
)

// RelayEnoughPeers determines if there are enough peers to publish a message on a topic
func RelayEnoughPeers(topic string) string {
	response, err := library.RelayEnoughPeers(topic)
	return prepareJSONResponse(response, err)
}

// RelayPublish publishes a message using waku relay and returns the message ID
func RelayPublish(messageJSON string, topic string, ms int) string {
	hash, err := library.RelayPublish(messageJSON, topic, ms)
	return prepareJSONResponse(hash, err)
}

// RelaySubscribe subscribes to a WakuRelay topic.
func RelaySubscribe(topic string) string {
	err := library.RelaySubscribe(topic)
	return makeJSONResponse(err)
}

// RelayTopics returns a list of pubsub topics the node is subscribed to in WakuRelay
func RelayTopics() string {
	topics, err := library.RelayTopics()
	return prepareJSONResponse(topics, err)
}

// RelayUnsubscribe closes the pubsub subscription to a pubsub topic
func RelayUnsubscribe(topic string) string {
	err := library.RelayUnsubscribe(topic)
	return makeJSONResponse(err)
}
