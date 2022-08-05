package main

import (
	"C"

	mobile "github.com/status-im/go-waku/mobile"
)

//export waku_filter_subscribe
// Creates a subscription to a light node matching a content filter and, optionally, a pubSub topic.
// filterJSON must contain a JSON with this format:
// {
// 	 "contentFilters": [ // mandatory
//     {
//       "contentTopic": "the content topic"
//     }, ...
//   ],
//   "topic": "the pubsub topic" // optional
// }
// peerID should contain the ID of a peer supporting the filter protocol. Use NULL to automatically select a node
// If ms is greater than 0, the subscription must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
func waku_filter_subscribe(filterJSON *C.char, peerID *C.char, ms C.int) *C.char {
	response := mobile.FilterSubscribe(C.GoString(filterJSON), C.GoString(peerID), int(ms))
	return C.CString(response)
}

//export waku_filter_unsubscribe
// Removes subscriptions in a light node matching a content filter and, optionally, a pubSub topic.
// filterJSON must contain a JSON with this format:
// {
// 	 "contentFilters": [ // mandatory
//     {
//       "contentTopic": "the content topic"
//     }, ...
//   ],
//   "topic": "the pubsub topic" // optional
// }
// If ms is greater than 0, the subscription must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
func waku_filter_unsubscribe(filterJSON *C.char, ms C.int) *C.char {
	response := mobile.FilterUnsubscribe(C.GoString(filterJSON), int(ms))
	return C.CString(response)
}
