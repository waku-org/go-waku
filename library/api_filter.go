package main

import (
	"C"

	mobile "github.com/waku-org/go-waku/mobile"
)

// Creates a subscription to a filter full node matching a content filter.
// filterJSON must contain a JSON with this format:
//
//		{
//		  "pubsubTopic": "the pubsub topic" // mandatory
//	      "contentTopics": ["the content topic"] // mandatory, at least one required, with a max of 10
//		}
//
// peerID should contain the ID of a peer supporting the filter protocol. Use NULL to automatically select a node
// If ms is greater than 0, the subscription must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
// It returns a json object containing the peerID to which we are subscribed to and the details of the subscription
//
//export waku_filter_subscribe
func waku_filter_subscribe(filterJSON *C.char, peerID *C.char, ms C.int) *C.char {
	response := mobile.FilterSubscribe(C.GoString(filterJSON), C.GoString(peerID), int(ms))
	return C.CString(response)
}

// Used to know if a service node has an active subscription for this client
// peerID should contain the ID of a peer we are subscribed to, supporting the filter protocol
// If ms is greater than 0, the subscription must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
//
//export waku_filter_ping
func waku_filter_ping(peerID *C.char, ms C.int) *C.char {
	response := mobile.FilterPing(C.GoString(peerID), int(ms))
	return C.CString(response)
}

// Sends a requests to a service node to stop pushing messages matching this filter to this client.
// It might be used to modify an existing subscription by providing a subset of the original filter
// criteria
//
//		{
//		  "pubsubTopic": "the pubsub topic" // mandatory
//	      "contentTopics": ["the content topic"] // mandatory, at least one required, with a max of 10
//		}
//
// peerID should contain the ID of a peer this client is subscribed to.
// If ms is greater than 0, the subscription must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
//
//export waku_filter_unsubscribe
func waku_filter_unsubscribe(filterJSON *C.char, peerID *C.char, ms C.int) *C.char {
	response := mobile.FilterUnsubscribe(C.GoString(filterJSON), C.GoString(peerID), int(ms))
	return C.CString(response)
}

// Sends a requests to a service node (or all service nodes) to stop pushing messages
// peerID should contain the ID of a peer this client is subscribed to, or can be NULL to
// stop all active subscriptions
// If ms is greater than 0, the subscription must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
//
//export waku_filter_unsubscribe_all
func waku_filter_unsubscribe_all(peerID *C.char, ms C.int) *C.char {
	response := mobile.FilterUnsubscribeAll(C.GoString(peerID), int(ms))
	return C.CString(response)
}
