package gowaku

import "github.com/waku-org/go-waku/library"

func LightpushPublish(messageJSON string, topic string, peerID string, ms int) string {
	response, err := library.LightpushPublish(messageJSON, topic, peerID, ms)
	return PrepareJSONResponse(response, err)
}

func LightpushPublishEncodeAsymmetric(messageJSON string, topic string, peerID string, publicKey string, optionalSigningKey string, ms int) string {
	response, err := library.LightpushPublishEncodeAsymmetric(messageJSON, topic, peerID, publicKey, optionalSigningKey, ms)
	return PrepareJSONResponse(response, err)
}

func LightpushPublishEncodeSymmetric(messageJSON string, topic string, peerID string, symmetricKey string, optionalSigningKey string, ms int) string {
	response, err := library.LightpushPublishEncodeSymmetric(messageJSON, topic, peerID, symmetricKey, optionalSigningKey, ms)
	return PrepareJSONResponse(response, err)
}
