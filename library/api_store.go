package main

import (
	"C"

	mobile "github.com/status-im/go-waku/mobile"
)

//export waku_store_query
// Query historic messages using waku store protocol.
// queryJSON must contain a valid json string with the following format:
// {
// 	"pubsubTopic": "...", // optional string
//  "startTime": 1234, // optional, unix epoch time in nanoseconds
//  "endTime": 1234, // optional, unix epoch time in nanoseconds
//  "contentFilters": [ // optional
// 	    {
//	       "contentTopic": "..."
//      }, ...
//  ],
//  "pagingOptions": {// optional pagination information
//      "pageSize": 40, // number
// 		"cursor": { // optional
//			"digest": ...,
//			"receiverTime": ...,
//			"senderTime": ...,
//			"pubsubTopic" ...,
//      }
//		"forward": true, // sort order
//  }
// }
// peerID should contain the ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
func waku_store_query(queryJSON *C.char, peerID *C.char, ms C.int) *C.char {
	response := mobile.StoreQuery(C.GoString(queryJSON), C.GoString(peerID), int(ms))
	return C.CString(response)
}
