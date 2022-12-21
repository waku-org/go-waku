package gowaku

import (
	"context"
	"errors"
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

func StartDiscoveryV5() string {
	if wakuNode == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}
	if wakuNode.DiscV5() == nil {
		return MakeJSONResponse(errors.New("DiscV5 is not mounted"))
	}
	err := wakuNode.DiscV5().Start(context.Background())
	return MakeJSONResponse(err)
}

func StopDiscoveryV5() string {
	if wakuNode == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}
	if wakuNode.DiscV5() == nil {
		return MakeJSONResponse(errors.New("DiscV5 is not mounted"))
	}
	wakuNode.DiscV5().Stop()
	return MakeJSONResponse(nil)
}
