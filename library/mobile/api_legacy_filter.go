package gowaku

import (
	"github.com/waku-org/go-waku/library"
)

// Deprecated: Use FilterSubscribe instead
func LegacyFilterSubscribe(filterJSON string, peerID string, ms int) string {
	err := library.LegacyFilterSubscribe(filterJSON, peerID, ms)
	return MakeJSONResponse(err)
}

// Deprecated: Use FilterUnsubscribe or FilterUnsubscribeAll instead
func LegacyFilterUnsubscribe(filterJSON string, ms int) string {
	err := library.LegacyFilterUnsubscribe(filterJSON, ms)
	return MakeJSONResponse(err)
}
