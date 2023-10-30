package library

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

// RelayEnoughPeers determines if there are enough peers to publish a message on a topic
func RelayEnoughPeers(topic string) (bool, error) {
	if wakuState.node == nil {
		return false, errWakuNodeNotReady
	}

	topicToCheck := protocol.DefaultPubsubTopic().String()
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

	return relayPublish(msg, topic, int(ms))
}

func relaySubscribe(filterJSON string) error {
	cf, err := toContentFilter(filterJSON)
	if err != nil {
		return err
	}

	subscriptions, err := wakuState.node.Relay().Subscribe(context.Background(), cf)
	if err != nil {
		return err
	}

	for _, sub := range subscriptions {
		go func(subscription *relay.Subscription) {
			for envelope := range subscription.Ch {
				send("message", toSubscriptionMessage(envelope))
			}
		}(sub)
	}

	return nil
}

// RelaySubscribe subscribes to a WakuRelay topic.
func RelaySubscribe(contentFilterJSON string) error {
	if wakuState.node == nil {
		return errWakuNodeNotReady
	}

	return relaySubscribe(contentFilterJSON)
}

// RelayTopics returns a list of pubsub topics the node is subscribed to in WakuRelay
func RelayTopics() (string, error) {
	if wakuState.node == nil {
		return "", errWakuNodeNotReady
	}

	return marshalJSON(wakuState.node.Relay().Topics())
}

// RelayUnsubscribe closes the pubsub subscription to a pubsub topic
func RelayUnsubscribe(contentFilterJSON string) error {
	cf, err := toContentFilter(contentFilterJSON)
	if err != nil {
		return err
	}

	if wakuState.node == nil {
		return errWakuNodeNotReady
	}

	return wakuState.node.Relay().Unsubscribe(context.Background(), cf)
}
