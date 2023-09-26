package main

import "C"
import "github.com/waku-org/go-waku/library"

// Determine if there are enough peers to publish a message on a topic. Use NULL
// to verify the number of peers in the default pubsub topic
//
//export waku_relay_enough_peers
func waku_relay_enough_peers(topic *C.char) *C.char {
	return singleFnExec(func() (any, error) {
		result, err := library.RelayEnoughPeers(C.GoString(topic))
		if result {
			return "true", err
		}
		return "false", err
	})
}

// Publish a message using waku relay and returns the message ID. Use NULL for topic to use the default pubsub topic
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned.
//
//export waku_relay_publish
func waku_relay_publish(messageJSON *C.char, topic *C.char, ms C.int) *C.char {
	return singleFnExec(func() (any, error) {
		return library.RelayPublish(C.GoString(messageJSON), C.GoString(topic), int(ms))
	})
}

// Subscribe to a WakuRelay topic. Set the topic to NULL to subscribe
// to the default topic. Returns a json response. When a message is received,
// a "message" event is emitted containing the message and pubsub topic in which
// the message was received
//
//export waku_relay_subscribe
func waku_relay_subscribe(topic *C.char) *C.char {
	err := library.RelaySubscribe(C.GoString(topic))
	return execErrCB(err)
}

// Returns a json response with the list of pubsub topics the node
// is subscribed to in WakuRelay
//
//export waku_relay_topics
func waku_relay_topics() *C.char {
	return singleFnExec(func() (any, error) {
		return library.RelayTopics()
	})
}

// Closes the pubsub subscription to a pubsub topic
//
//export waku_relay_unsubscribe
func waku_relay_unsubscribe(topic *C.char) *C.char {
	err := library.RelayUnsubscribe(C.GoString(topic))
	return execErrCB(err)
}
