package gowaku

import (
	"context"
	"time"

	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

var relaySubscriptions map[string]*relay.Subscription = make(map[string]*relay.Subscription)
var relaySubsMutex sync.Mutex

func RelayEnoughPeers(topic string) string {
	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	topicToCheck := protocol.DefaultPubsubTopic().String()
	if topic != "" {
		topicToCheck = topic
	}

	return PrepareJSONResponse(wakuState.node.Relay().EnoughPeersToPublishToTopic(topicToCheck), nil)
}

func relayPublish(msg *pb.WakuMessage, pubsubTopic string, ms int) (string, error) {
	if wakuState.node == nil {
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

	hash, err := wakuState.node.Relay().PublishToTopic(ctx, msg, pubsubTopic)
	return hexutil.Encode(hash), err
}

func RelayPublish(messageJSON string, topic string, ms int) string {
	msg, err := wakuMessage(messageJSON)
	if err != nil {
		return MakeJSONResponse(err)
	}

	hash, err := relayPublish(msg, getTopic(topic), int(ms))
	return PrepareJSONResponse(hash, err)
}

func RelayPublishEncodeAsymmetric(messageJSON string, topic string, publicKey string, optionalSigningKey string, ms int) string {
	msg, err := wakuMessageAsymmetricEncoding(messageJSON, publicKey, optionalSigningKey)
	if err != nil {
		return MakeJSONResponse(err)
	}

	hash, err := relayPublish(msg, getTopic(topic), int(ms))

	return PrepareJSONResponse(hash, err)
}

func RelayPublishEncodeSymmetric(messageJSON string, topic string, symmetricKey string, optionalSigningKey string, ms int) string {
	msg, err := wakuMessageSymmetricEncoding(messageJSON, symmetricKey, optionalSigningKey)
	if err != nil {
		return MakeJSONResponse(err)
	}

	hash, err := relayPublish(msg, getTopic(topic), int(ms))

	return PrepareJSONResponse(hash, err)
}

func RelaySubscribe(topic string) string {
	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	topicToSubscribe := getTopic(topic)

	relaySubsMutex.Lock()
	defer relaySubsMutex.Unlock()

	_, ok := relaySubscriptions[topicToSubscribe]
	if ok {
		return MakeJSONResponse(nil)
	}

	subscription, err := wakuState.node.Relay().SubscribeToTopic(context.Background(), topicToSubscribe)
	if err != nil {
		return MakeJSONResponse(err)
	}

	relaySubscriptions[topicToSubscribe] = subscription

	go func(subscription *relay.Subscription) {
		for envelope := range subscription.C {
			send("message", toSubscriptionMessage(envelope))
		}
	}(subscription)

	return MakeJSONResponse(nil)
}

func RelayUnsubscribe(topic string) string {
	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	topicToUnsubscribe := getTopic(topic)

	relaySubsMutex.Lock()
	defer relaySubsMutex.Unlock()

	subscription, ok := relaySubscriptions[topicToUnsubscribe]
	if ok {
		return MakeJSONResponse(nil)
	}

	subscription.Unsubscribe()

	delete(relaySubscriptions, topicToUnsubscribe)

	err := wakuState.node.Relay().Unsubscribe(context.Background(), topicToUnsubscribe)
	if err != nil {
		return MakeJSONResponse(err)
	}

	return MakeJSONResponse(nil)
}
