package main

/*
#include <cgo_utils.h>
*/
import "C"
import (
	"unsafe"

	"github.com/waku-org/go-waku/library"
)

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
func waku_legacy_filter_subscribe(ctx unsafe.Pointer, filterJSON *C.char, peerID *C.char, ms C.int, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		onError(err, cb, userData)
	}

	err = library.LegacyFilterSubscribe(instance, C.GoString(filterJSON), C.GoString(peerID), int(ms))
	return onError(err, cb, userData)
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
func waku_legacy_filter_unsubscribe(ctx unsafe.Pointer, filterJSON *C.char, ms C.int, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		onError(err, cb, userData)
	}

	err = library.LegacyFilterUnsubscribe(instance, C.GoString(filterJSON), int(ms))
	return onError(err, cb, userData)
}
