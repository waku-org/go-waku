package main

import (
	"C"

	mobile "github.com/waku-org/go-waku/mobile"
)

// Returns a list of objects containing the peerID, enr and multiaddresses for each node found
//  given a url to a DNS discoverable ENR tree
// The nameserver can optionally be specified to resolve the enrtree url. Otherwise NULL or
// empty to automatically use the default system dns.
// If ms is greater than 0, the subscription must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
//
//export waku_dns_discovery
func waku_dns_discovery(url *C.char, nameserver *C.char, ms C.int) *C.char {
	response := mobile.DnsDiscovery(C.GoString(url), C.GoString(nameserver), int(ms))
	return C.CString(response)
}
