package gowaku

import (
	"github.com/waku-org/go-waku/library"
)

func DnsDiscovery(url string, nameserver string, ms int) string {
	response, err := library.DnsDiscovery(url, nameserver, ms)
	return PrepareJSONResponse(response, err)
}

func StartDiscoveryV5() string {
	err := library.StartDiscoveryV5()
	return MakeJSONResponse(err)
}

func StopDiscoveryV5() string {
	err := library.StopDiscoveryV5()
	return MakeJSONResponse(err)
}

func SetBootnodes(bootnodes string) string {
	err := library.SetBootnodes(bootnodes)
	return MakeJSONResponse(err)
}
