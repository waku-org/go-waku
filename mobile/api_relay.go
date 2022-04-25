package gowaku

import (
	"context"
	"time"

	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
)

var subscriptions map[string]*relay.Subscription = make(map[string]*relay.Subscription)
var mutex sync.Mutex

func RelayEnoughPeers(topic string) string {
	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
	}

	topicToCheck := protocol.DefaultPubsubTopic().String()
	if topic != "" {
		topicToCheck = topic
	}

	return prepareJSONResponse(wakuNode.Relay().EnoughPeersToPublishToTopic(topicToCheck), nil)
}

func relayPublish(msg pb.WakuMessage, pubsubTopic string, ms int) (string, error) {
	if wakuNode == nil {
		return "", errWakuNodeNotReady
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	hash, err := wakuNode.Relay().PublishToTopic(ctx, &msg, pubsubTopic)
	return hexutil.Encode(hash), err
}

func RelayPublish(messageJSON string, topic string, ms int) string {
	msg, err := wakuMessage(messageJSON)
	if err != nil {
		return makeJSONResponse(err)
	}

	hash, err := relayPublish(msg, getTopic(topic), int(ms))
	return prepareJSONResponse(hash, err)
}

func RelayPublishEncodeAsymmetric(messageJSON string, topic string, publicKey string, optionalSigningKey string, ms int) string {
	msg, err := wakuMessageAsymmetricEncoding(messageJSON, publicKey, optionalSigningKey)
	if err != nil {
		return makeJSONResponse(err)
	}

	hash, err := relayPublish(msg, getTopic(topic), int(ms))

	return prepareJSONResponse(hash, err)
}

func RelayPublishEncodeSymmetric(messageJSON string, topic string, symmetricKey string, optionalSigningKey string, ms int) string {
	msg, err := wakuMessageSymmetricEncoding(messageJSON, symmetricKey, optionalSigningKey)
	if err != nil {
		return makeJSONResponse(err)
	}

	hash, err := relayPublish(msg, getTopic(topic), int(ms))

	return prepareJSONResponse(hash, err)
}

func RelaySubscribe(topic string) string {
	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
	}

	topicToSubscribe := getTopic(topic)

	mutex.Lock()
	defer mutex.Unlock()

	_, ok := subscriptions[topicToSubscribe]
	if ok {
		return makeJSONResponse(nil)
	}

	subscription, err := wakuNode.Relay().SubscribeToTopic(context.Background(), topicToSubscribe)
	if err != nil {
		return makeJSONResponse(err)
	}

	subscriptions[topicToSubscribe] = subscription

	go func(subscription *relay.Subscription) {
		for envelope := range subscription.C {
			send("message", toSubscriptionMessage(envelope))
		}
	}(subscription)

	return makeJSONResponse(nil)
}

func RelayUnsubscribe(topic string) string {
	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
	}

	topicToUnsubscribe := getTopic(topic)

	mutex.Lock()
	defer mutex.Unlock()

	subscription, ok := subscriptions[topicToUnsubscribe]
	if ok {
		return makeJSONResponse(nil)
	}

	subscription.Unsubscribe()

	delete(subscriptions, topicToUnsubscribe)

	err := wakuNode.Relay().Unsubscribe(context.Background(), topicToUnsubscribe)
	if err != nil {
		return makeJSONResponse(err)
	}

	return makeJSONResponse(nil)
}
