package gowaku

import (
	"github.com/waku-org/go-waku/library"
)

// DNSDiscovery executes dns discovery on an url and returns a list of nodes
func DNSDiscovery(url string, nameserver string, ms int) string {
	response, err := library.DNSDiscovery(url, nameserver, ms)
	return prepareJSONResponse(response, err)
}

// StartDiscoveryV5 starts discv5 discovery
func StartDiscoveryV5() string {
	err := library.StartDiscoveryV5()
	return makeJSONResponse(err)
}

// StopDiscoveryV5 stops discv5 discovery
func StopDiscoveryV5() string {
	err := library.StopDiscoveryV5()
	return makeJSONResponse(err)
}

// SetBootnodes is used to update the bootnodes receiving a JSON array of ENRs
func SetBootnodes(bootnodes string) string {
	err := library.SetBootnodes(bootnodes)
	return makeJSONResponse(err)
}
