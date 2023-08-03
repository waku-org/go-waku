package gowaku

import (
	"github.com/waku-org/go-waku/library"
)

// StoreQuery is used to retrieve historic messages using waku store protocol.
func StoreQuery(queryJSON string, peerID string, ms int) string {
	response, err := library.StoreQuery(queryJSON, peerID, ms)
	return prepareJSONResponse(response, err)
}

// StoreLocalQuery is used to retrieve historic messages stored in the localDB using waku store protocol.
func StoreLocalQuery(queryJSON string) string {
	response, err := library.StoreLocalQuery(queryJSON)
	return prepareJSONResponse(response, err)
}
