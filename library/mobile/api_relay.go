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

// RelayPublishEncodeAsymmetric publish a message encrypted with a secp256k1 public key using waku relay and returns the message ID
func RelayPublishEncodeAsymmetric(messageJSON string, topic string, publicKey string, optionalSigningKey string, ms int) string {
	hash, err := library.RelayPublishEncodeAsymmetric(messageJSON, topic, publicKey, optionalSigningKey, ms)
	return prepareJSONResponse(hash, err)
}

// RelayPublishEncodeSymmetric publishes a message encrypted with a 32 bytes symmetric key using waku relay and returns the message ID
func RelayPublishEncodeSymmetric(messageJSON string, topic string, symmetricKey string, optionalSigningKey string, ms int) string {
	hash, err := library.RelayPublishEncodeSymmetric(messageJSON, topic, symmetricKey, optionalSigningKey, ms)
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
