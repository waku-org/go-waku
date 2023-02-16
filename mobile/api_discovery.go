package gowaku

import (
	"context"
	"errors"
	"time"

	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
)

type DnsDiscoveryItem struct {
	PeerID    string   `json:"peerID,omitempty"`
	Addresses []string `json:"multiaddrs,omitempty"`
	ENR       string   `json:"enr,omitempty"`
}

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

	var response []DnsDiscoveryItem
	for _, n := range nodes {
		item := DnsDiscoveryItem{
			PeerID: n.PeerID.String(),
		}
		for _, addr := range n.Addresses {
			item.Addresses = append(item.Addresses, addr.String())
		}
		item.ENR = n.ENR.String()
		response = append(response, item)
	}

	return PrepareJSONResponse(response, nil)
}

func StartDiscoveryV5() string {
	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}
	if wakuState.node.DiscV5() == nil {
		return MakeJSONResponse(errors.New("DiscV5 is not mounted"))
	}
	err := wakuState.node.DiscV5().Start(context.Background())
	if err != nil {
		return MakeJSONResponse(err)
	}

	return MakeJSONResponse(nil)
}

func StopDiscoveryV5() string {
	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}
	if wakuState.node.DiscV5() == nil {
		return MakeJSONResponse(errors.New("DiscV5 is not mounted"))
	}
	wakuState.node.DiscV5().Stop()
	return MakeJSONResponse(nil)
}
