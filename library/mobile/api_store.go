package gowaku

import (
	"github.com/waku-org/go-waku/library"
)

func StoreQuery(queryJSON string, peerID string, ms int) string {
	response, err := library.StoreQuery(queryJSON, peerID, ms)
	return PrepareJSONResponse(response, err)
}

func StoreLocalQuery(queryJSON string) string {
	response, err := library.StoreLocalQuery(queryJSON)
	return PrepareJSONResponse(response, err)
}
