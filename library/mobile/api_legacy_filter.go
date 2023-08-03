package gowaku

import (
	"github.com/waku-org/go-waku/library"
)

// LegacyFilterSubscribe is used to create a subscription to a filter node to receive messages
// Deprecated: Use FilterSubscribe instead
func LegacyFilterSubscribe(filterJSON string, peerID string, ms int) string {
	err := library.LegacyFilterSubscribe(filterJSON, peerID, ms)
	return makeJSONResponse(err)
}

// LegacyFilterUnsubscribe is used to remove a filter criteria from an active subscription with a filter node
// Deprecated: Use FilterUnsubscribe or FilterUnsubscribeAll instead
func LegacyFilterUnsubscribe(filterJSON string, ms int) string {
	err := library.LegacyFilterUnsubscribe(filterJSON, ms)
	return makeJSONResponse(err)
}
