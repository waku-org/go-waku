package gowaku

import "github.com/waku-org/go-waku/library"

// LightpushPublish is used to publish a WakuMessage in a pubsub topic using Lightpush protocol
func LightpushPublish(messageJSON string, topic string, peerID string, ms int) string {
	response, err := library.LightpushPublish(messageJSON, topic, peerID, ms)
	return prepareJSONResponse(response, err)
}

// LightpushPublishEncodeAsymmetric is used to publish a WakuMessage in a pubsub topic using Lightpush protocol, and encrypting the message with some public key
func LightpushPublishEncodeAsymmetric(messageJSON string, topic string, peerID string, publicKey string, optionalSigningKey string, ms int) string {
	response, err := library.LightpushPublishEncodeAsymmetric(messageJSON, topic, peerID, publicKey, optionalSigningKey, ms)
	return prepareJSONResponse(response, err)
}

// LightpushPublishEncodeSymmetric is used to publish a WakuMessage in a pubsub topic using Lightpush protocol, and encrypting the message with a symmetric key
func LightpushPublishEncodeSymmetric(messageJSON string, topic string, peerID string, symmetricKey string, optionalSigningKey string, ms int) string {
	response, err := library.LightpushPublishEncodeSymmetric(messageJSON, topic, peerID, symmetricKey, optionalSigningKey, ms)
	return prepareJSONResponse(response, err)
}
