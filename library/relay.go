package library

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

// RelayEnoughPeers determines if there are enough peers to publish a message on a topic
func RelayEnoughPeers(topic string) (bool, error) {
	if wakuState.node == nil {
		return false, errWakuNodeNotReady
	}

	topicToCheck := protocol.DefaultPubsubTopic{}.String()
	if topic != "" {
		topicToCheck = topic
	}

	return wakuState.node.Relay().EnoughPeersToPublishToTopic(topicToCheck), nil
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

// RelayPublish publishes a message using waku relay and returns the message ID
func RelayPublish(messageJSON string, topic string, ms int) (string, error) {
	msg, err := wakuMessage(messageJSON)
	if err != nil {
		return "", err
	}

	return relayPublish(msg, getTopic(topic), int(ms))
}

func relaySubscribe(topic string) error {
	topicToSubscribe := getTopic(topic)

	relaySubsMutex.Lock()
	defer relaySubsMutex.Unlock()

	_, ok := relaySubscriptions[topicToSubscribe]
	if ok {
		return nil
	}

	subscription, err := wakuState.node.Relay().Subscribe(context.Background(), protocol.NewContentFilter(topicToSubscribe))
	if err != nil {
		return err
	}

	relaySubscriptions[topicToSubscribe] = subscription[0]

	go func(subscription *relay.Subscription) {
		for envelope := range subscription.Ch {
			send("message", toSubscriptionMessage(envelope))
		}
	}(subscription[0])

	return nil
}

// RelaySubscribe subscribes to a WakuRelay topic.
func RelaySubscribe(topic string) error {
	if wakuState.node == nil {
		return errWakuNodeNotReady
	}

	return relaySubscribe(topic)
}

// RelayTopics returns a list of pubsub topics the node is subscribed to in WakuRelay
func RelayTopics() (string, error) {
	if wakuState.node == nil {
		return "", errWakuNodeNotReady
	}

	return marshalJSON(wakuState.node.Relay().Topics())
}

// RelayUnsubscribe closes the pubsub subscription to a pubsub topic
func RelayUnsubscribe(topic string) error {
	if wakuState.node == nil {
		return errWakuNodeNotReady
	}

	topicToUnsubscribe := getTopic(topic)

	relaySubsMutex.Lock()
	defer relaySubsMutex.Unlock()

	subscription, ok := relaySubscriptions[topicToUnsubscribe]
	if ok {
		return nil
	}

	subscription.Unsubscribe()

	delete(relaySubscriptions, topicToUnsubscribe)

	return wakuState.node.Relay().Unsubscribe(context.Background(), protocol.NewContentFilter(topicToUnsubscribe))
}
