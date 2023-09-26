package main

import "C"
import "github.com/waku-org/go-waku/library"

// Publish a message using waku lightpush. Use NULL for topic to use the default pubsub topic..
// peerID should contain the ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
//
//export waku_lightpush_publish
func waku_lightpush_publish(messageJSON *C.char, topic *C.char, peerID *C.char, ms C.int) *C.char {
	return singleFnExec(func() (any, error) {
		return library.LightpushPublish(C.GoString(messageJSON), C.GoString(topic), C.GoString(peerID), int(ms))
	})
}
