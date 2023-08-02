package main

/*
#include <cgo_utils.h>
*/
import "C"
import "github.com/waku-org/go-waku/library"

// Creates a subscription to a light node matching a content filter and, optionally, a pubSub topic.
// filterJSON must contain a JSON with this format:
//
//	{
//		 "contentFilters": [ // mandatory
//	    {
//	      "contentTopic": "the content topic"
//	    }, ...
//	  ],
//	  "pubsubTopic": "the pubsub topic" // optional
//	}
//
// peerID should contain the ID of a peer supporting the filter protocol. Use NULL to automatically select a node
// If ms is greater than 0, the subscription must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
//
//export waku_legacy_filter_subscribe
func waku_legacy_filter_subscribe(filterJSON *C.char, peerID *C.char, ms C.int, onErrCb C.WakuCallBack) C.int {
	err := library.LegacyFilterSubscribe(C.GoString(filterJSON), C.GoString(peerID), int(ms))
	return execErrCB(onErrCb, err)
}

// Removes subscriptions in a light node matching a content filter and, optionally, a pubSub topic.
// filterJSON must contain a JSON with this format:
//
//	{
//		 "contentFilters": [ // mandatory
//	    {
//	      "contentTopic": "the content topic"
//	    }, ...
//	  ],
//	  "pubsubTopic": "the pubsub topic" // optional
//	}
//
// If ms is greater than 0, the subscription must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
//
//export waku_legacy_filter_unsubscribe
func waku_legacy_filter_unsubscribe(filterJSON *C.char, ms C.int, onErrCb C.WakuCallBack) C.int {
	err := library.LegacyFilterUnsubscribe(C.GoString(filterJSON), int(ms))
	return execErrCB(onErrCb, err)
}
