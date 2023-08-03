package gowaku

import (
	"github.com/waku-org/go-waku/library"
)

// FilterSubscribe is used to create a subscription to a filter node to receive messages
func FilterSubscribe(filterJSON string, peerID string, ms int) string {
	response, err := library.FilterSubscribe(filterJSON, peerID, ms)
	return prepareJSONResponse(response, err)
}

// FilterPing is used to determine if a peer has an active subscription
func FilterPing(peerID string, ms int) string {
	err := library.FilterPing(peerID, ms)
	return makeJSONResponse(err)
}

// FilterUnsubscribe is used to remove a filter criteria from an active subscription with a filter node
func FilterUnsubscribe(filterJSON string, peerID string, ms int) string {
	err := library.FilterUnsubscribe(filterJSON, peerID, ms)
	return makeJSONResponse(err)
}

// FilterUnsubscribeAll is used to remove an active subscription to a peer. If no peerID is defined, it will stop all active filter subscriptions
func FilterUnsubscribeAll(peerID string, ms int) string {
	response, err := library.FilterUnsubscribeAll(peerID, ms)
	return prepareJSONResponse(response, err)
}
