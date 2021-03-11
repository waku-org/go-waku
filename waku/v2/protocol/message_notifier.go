// The Message Notification system is a method to notify various protocols
// running on a node when a new message was received.
//
// Protocols can subscribe to messages of specific topics, then when one is received
// The notification handler function will be called.

package protocol

import (
	"sync"
)

type MessageNotificationHandler func(topic string, msg *WakuMessage)

type MessageNotificationSubscriptionIdentifier string

type MessageNotificationSubscription struct {
	topics  []string // @TODO TOPIC (?)
	handler MessageNotificationHandler
}

type MessageNotificationSubscriptions map[string]MessageNotificationSubscription

func (subscriptions MessageNotificationSubscriptions) subscribe(name string, subscription MessageNotificationSubscription) {
	subscriptions[name] = subscription
}

func Init(topics []string, handler MessageNotificationHandler) MessageNotificationSubscription {
	result := MessageNotificationSubscription{}
	result.topics = topics
	result.handler = handler
	return result
}

func containsMatch(lhs []string, rhs []string) bool {
	for _, l := range lhs {
		for _, r := range rhs {
			if l == r {
				return true
			}
		}
	}
	return false
}

func (subscriptions MessageNotificationSubscriptions) notify(topic string, msg *WakuMessage) {
	var wg sync.WaitGroup

	for _, subscription := range subscriptions {
		// @TODO WILL NEED TO CHECK SUBTOPICS IN FUTURE FOR WAKU TOPICS NOT LIBP2P ONES

		found := false
		for _, subscriptionTopic := range subscription.topics {
			if subscriptionTopic == topic {
				found = true
				break
			}
		}

		if !found {
			continue
		}

		wg.Add(1)
		go func(subs MessageNotificationSubscription) {
			subs.handler(topic, msg)
			wg.Done()
		}(subscription)
	}

	wg.Wait()
}
