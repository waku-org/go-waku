package gowaku

import "github.com/waku-org/go-waku/library"

// LightpushPublish is used to publish a WakuMessage in a pubsub topic using Lightpush protocol
func LightpushPublish(messageJSON string, topic string, peerID string, ms int) string {
	response, err := library.LightpushPublish(messageJSON, topic, peerID, ms)
	return prepareJSONResponse(response, err)
}
