package gowaku

import (
	"context"
	"time"

	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
)

func DnsDiscovery(url string, nameserver string, ms int) string {
	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	var dnsDiscOpt []dnsdisc.DnsDiscoveryOption
	if nameserver != "" {
		dnsDiscOpt = append(dnsDiscOpt, dnsdisc.WithNameserver(nameserver))
	}

	nodes, err := dnsdisc.RetrieveNodes(ctx, url, dnsDiscOpt...)
	if err != nil {
		return MakeJSONResponse(err)
	}

	var ma []string
	for _, n := range nodes {
		for _, addr := range n.Addresses {
			ma = append(ma, addr.String())
		}
	}

	return PrepareJSONResponse(ma, nil)
}
