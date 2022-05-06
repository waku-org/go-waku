package main

import (
	"C"

	mobile "github.com/status-im/go-waku/mobile"
)

//export waku_lightpush_publish
// Publish a message using waku lightpush. Use NULL for topic to use the default pubsub topic..
// peerID should contain the ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
func waku_lightpush_publish(messageJSON *C.char, topic *C.char, peerID *C.char, ms C.int) *C.char {
	response := mobile.LightpushPublish(C.GoString(messageJSON), C.GoString(topic), C.GoString(peerID), int(ms))
	return C.CString(response)
}

//export waku_lightpush_publish_enc_asymmetric
// Publish a message encrypted with a secp256k1 public key using waku lightpush. Use NULL for topic to use the default pubsub topic.
// peerID should contain the ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node
// publicKey must be a hex string prefixed with "0x" containing a valid secp256k1 public key.
// optionalSigningKey is an optional hex string prefixed with "0x" containing a valid secp256k1 private key for signing the message. Use NULL otherwise
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned.
func waku_lightpush_publish_enc_asymmetric(messageJSON *C.char, topic *C.char, peerID *C.char, publicKey *C.char, optionalSigningKey *C.char, ms C.int) *C.char {
	response := mobile.LightpushPublishEncodeAsymmetric(C.GoString(messageJSON), C.GoString(topic), C.GoString(peerID), C.GoString(publicKey), C.GoString(optionalSigningKey), int(ms))
	return C.CString(response)
}

//export waku_lightpush_publish_enc_symmetric
// Publish a message encrypted with a 32 bytes symmetric key using waku relay. Use NULL for topic to use the default pubsub topic.
// peerID should contain the ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node
// symmetricKey must be a hex string prefixed with "0x" containing a 32 bytes symmetric key
// optionalSigningKey is an optional hex string prefixed with "0x" containing a valid secp256k1 private key for signing the message. Use NULL otherwise
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned.
func waku_lightpush_publish_enc_symmetric(messageJSON *C.char, topic *C.char, peerID *C.char, symmetricKey *C.char, optionalSigningKey *C.char, ms C.int) *C.char {
	response := mobile.LightpushPublishEncodeSymmetric(C.GoString(messageJSON), C.GoString(topic), C.GoString(peerID), C.GoString(symmetricKey), C.GoString(optionalSigningKey), int(ms))
	return C.CString(response)
}
