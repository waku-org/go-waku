package gowaku

import (
	"github.com/waku-org/go-waku/library"
)

func FilterSubscribe(filterJSON string, peerID string, ms int) string {
	response, err := library.FilterSubscribe(filterJSON, peerID, ms)
	return PrepareJSONResponse(response, err)
}

func FilterPing(peerID string, ms int) string {
	err := library.FilterPing(peerID, ms)
	return MakeJSONResponse(err)
}

func FilterUnsubscribe(filterJSON string, peerID string, ms int) string {
	err := library.FilterUnsubscribe(filterJSON, peerID, ms)
	return MakeJSONResponse(err)
}

func FilterUnsubscribeAll(peerID string, ms int) string {
	response, err := library.FilterUnsubscribeAll(peerID, ms)
	return PrepareJSONResponse(response, err)
}
