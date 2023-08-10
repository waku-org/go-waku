package main

/*
#include <cgo_utils.h>
*/
import "C"
import "github.com/waku-org/go-waku/library"

// Query historic messages using waku store protocol.
// queryJSON must contain a valid json string with the following format:
//
//	{
//		"pubsubTopic": "...", // optional string
//	 "startTime": 1234, // optional, unix epoch time in nanoseconds
//	 "endTime": 1234, // optional, unix epoch time in nanoseconds
//	 "contentFilters": [ // optional
//			{
//		      contentTopic: "contentTopic1"
//			}, ...
//	 ],
//	 "pagingOptions": {// optional pagination information
//	     "pageSize": 40, // number
//			"cursor": { // optional
//				"digest": ...,
//				"receiverTime": ...,
//				"senderTime": ...,
//				"pubsubTopic" ...,
//	     }
//			"forward": true, // sort order
//	 }
//	}
//
// If a non empty cursor is returned, this function should be executed again, setting  the `cursor` attribute with the cursor returned in the response
// peerID should contain the ID of a peer supporting the store protocol. Use NULL to automatically select a node
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
//
//export waku_store_query
func waku_store_query(queryJSON *C.char, peerID *C.char, ms C.int, onOkCb C.WakuCallBack, onErrCb C.WakuCallBack) C.int {
	return singleFnExec(func() (string, error) {
		return library.StoreQuery(C.GoString(queryJSON), C.GoString(peerID), int(ms))
	}, onOkCb, onErrCb)
}

// Query historic messages stored in the localDB using waku store protocol.
// queryJSON must contain a valid json string with the following format:
//
//	{
//		"pubsubTopic": "...", // optional string
//	 "startTime": 1234, // optional, unix epoch time in nanoseconds
//	 "endTime": 1234, // optional, unix epoch time in nanoseconds
//	 "contentFilters": [ // optional
//			{
//		      contentTopic: "contentTopic1"
//			}, ...
//	 ],
//	 "pagingOptions": {// optional pagination information
//	     "pageSize": 40, // number
//			"cursor": { // optional
//				"digest": ...,
//				"receiverTime": ...,
//				"senderTime": ...,
//				"pubsubTopic" ...,
//	     }
//			"forward": true, // sort order
//	 }
//	}
//
// If a non empty cursor is returned, this function should be executed again, setting  the `cursor` attribute with the cursor returned in the response
// Requires the `store` option to be passed when setting up the initial configuration
//
//export waku_store_local_query
func waku_store_local_query(queryJSON *C.char, onOkCb C.WakuCallBack, onErrCb C.WakuCallBack) C.int {
	return singleFnExec(func() (string, error) {
		return library.StoreLocalQuery(C.GoString(queryJSON))
	}, onOkCb, onErrCb)
}
