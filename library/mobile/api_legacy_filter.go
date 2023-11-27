package gowaku

import (
	"github.com/waku-org/go-waku/library"
)

// LegacyFilterSubscribe is used to create a subscription to a filter node to receive messages
// Deprecated: Use FilterSubscribe instead
func LegacyFilterSubscribe(instanceID uint, filterJSON string, peerID string, ms int) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.LegacyFilterSubscribe(instance, filterJSON, peerID, ms)
	return makeJSONResponse(err)
}

// LegacyFilterUnsubscribe is used to remove a filter criteria from an active subscription with a filter node
// Deprecated: Use FilterUnsubscribe or FilterUnsubscribeAll instead
func LegacyFilterUnsubscribe(instanceID uint, filterJSON string, ms int) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.LegacyFilterUnsubscribe(instance, filterJSON, ms)
	return makeJSONResponse(err)
}
