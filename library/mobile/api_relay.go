package gowaku

import (
	"sync"

	"github.com/waku-org/go-waku/library"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

var relaySubscriptions map[string]*relay.Subscription = make(map[string]*relay.Subscription)
var relaySubsMutex sync.Mutex

func RelayEnoughPeers(topic string) string {
	response, err := library.RelayEnoughPeers(topic)
	return PrepareJSONResponse(response, err)
}

func RelayPublish(messageJSON string, topic string, ms int) string {
	hash, err := library.RelayPublish(messageJSON, topic, ms)
	return PrepareJSONResponse(hash, err)
}

func RelayPublishEncodeAsymmetric(messageJSON string, topic string, publicKey string, optionalSigningKey string, ms int) string {
	hash, err := library.RelayPublishEncodeAsymmetric(messageJSON, topic, publicKey, optionalSigningKey, ms)
	return PrepareJSONResponse(hash, err)
}

func RelayPublishEncodeSymmetric(messageJSON string, topic string, symmetricKey string, optionalSigningKey string, ms int) string {
	hash, err := library.RelayPublishEncodeSymmetric(messageJSON, topic, symmetricKey, optionalSigningKey, ms)
	return PrepareJSONResponse(hash, err)
}

func RelaySubscribe(topic string) string {
	err := library.RelaySubscribe(topic)
	return MakeJSONResponse(err)
}

func RelayTopics() string {
	topics, err := library.RelayTopics()
	return PrepareJSONResponse(topics, err)
}

func RelayUnsubscribe(topic string) string {
	err := library.RelayUnsubscribe(topic)
	return MakeJSONResponse(err)
}
