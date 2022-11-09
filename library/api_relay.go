package main

import (
	"C"

	mobile "github.com/waku-org/go-waku/mobile"
)

//export waku_relay_enough_peers
// Determine if there are enough peers to publish a message on a topic. Use NULL
// to verify the number of peers in the default pubsub topic
func waku_relay_enough_peers(topic *C.char) *C.char {
	response := mobile.RelayEnoughPeers(C.GoString(topic))
	return C.CString(response)
}

//export waku_relay_publish
// Publish a message using waku relay and returns the message ID. Use NULL for topic to use the default pubsub topic
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned.
func waku_relay_publish(messageJSON *C.char, topic *C.char, ms C.int) *C.char {
	response := mobile.RelayPublish(C.GoString(messageJSON), C.GoString(topic), int(ms))
	return C.CString(response)
}

//export waku_relay_publish_enc_asymmetric
// Publish a message encrypted with a secp256k1 public key using waku relay  and returns the message ID. Use NULL for topic to use the default pubsub topic.
// publicKey must be a hex string prefixed with "0x" containing a valid secp256k1 public key.
// optionalSigningKey is an optional hex string prefixed with "0x" containing a valid secp256k1 private key for signing the message. Use NULL otherwise
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned.
func waku_relay_publish_enc_asymmetric(messageJSON *C.char, topic *C.char, publicKey *C.char, optionalSigningKey *C.char, ms C.int) *C.char {
	response := mobile.RelayPublishEncodeAsymmetric(C.GoString(messageJSON), C.GoString(topic), C.GoString(publicKey), C.GoString(optionalSigningKey), int(ms))
	return C.CString(response)
}

//export waku_relay_publish_enc_symmetric
// Publish a message encrypted with a 32 bytes symmetric key using waku relay  and returns the message ID. Use NULL for topic to use the default pubsub topic.
// symmetricKey must be a hex string prefixed with "0x" containing a 32 bytes symmetric key
// optionalSigningKey is an optional hex string prefixed with "0x" containing a valid secp256k1 private key for signing the message. Use NULL otherwise
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned.
func waku_relay_publish_enc_symmetric(messageJSON *C.char, topic *C.char, symmetricKey *C.char, optionalSigningKey *C.char, ms C.int) *C.char {
	response := mobile.RelayPublishEncodeSymmetric(C.GoString(messageJSON), C.GoString(topic), C.GoString(symmetricKey), C.GoString(optionalSigningKey), int(ms))
	return C.CString(response)
}

//export waku_relay_subscribe
// Subscribe to a WakuRelay topic. Set the topic to NULL to subscribe
// to the default topic. Returns a json response. When a message is received,
// a "message" event is emitted containing the message and pubsub topic in which
// the message was received
func waku_relay_subscribe(topic *C.char) *C.char {
	response := mobile.RelaySubscribe(C.GoString(topic))
	return C.CString(response)
}

//export waku_relay_unsubscribe
// Closes the pubsub subscription to a pubsub topic
func waku_relay_unsubscribe(topic *C.char) *C.char {
	response := mobile.RelayUnsubscribe(C.GoString(topic))
	return C.CString(response)
}
