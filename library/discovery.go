package library

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
)

type DnsDiscoveryItem struct {
	PeerID    string   `json:"peerID"`
	Addresses []string `json:"multiaddrs"`
	ENR       string   `json:"enr,omitempty"`
}

func DnsDiscovery(url string, nameserver string, ms int) (string, error) {
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
		return "", err
	}

	var response []DnsDiscoveryItem
	for _, n := range nodes {
		item := DnsDiscoveryItem{
			PeerID: n.PeerID.String(),
		}
		for _, addr := range n.PeerInfo.Addrs {
			item.Addresses = append(item.Addresses, addr.String())
		}

		if n.ENR != nil {
			item.ENR = n.ENR.String()
		}

		response = append(response, item)
	}

	return MarshalJSON(response)
}

func StartDiscoveryV5() error {
	if wakuState.node == nil {
		return errWakuNodeNotReady
	}
	if wakuState.node.DiscV5() == nil {
		return errors.New("DiscV5 is not mounted")
	}
	return wakuState.node.DiscV5().Start(context.Background())
}

func StopDiscoveryV5() error {
	if wakuState.node == nil {
		return errWakuNodeNotReady
	}
	if wakuState.node.DiscV5() == nil {
		return errors.New("DiscV5 is not mounted")
	}
	wakuState.node.DiscV5().Stop()
	return nil
}

func SetBootnodes(bootnodes string) error {
	if wakuState.node == nil {
		return errWakuNodeNotReady
	}
	if wakuState.node.DiscV5() == nil {
		return errors.New("DiscV5 is not mounted")
	}

	var tmp []json.RawMessage
	if err := json.Unmarshal([]byte(bootnodes), &tmp); err != nil {
		return err
	}

	var enrList []string
	for _, el := range tmp {
		var enr string
		if err := json.Unmarshal(el, &enr); err != nil {
			return err
		}
		enrList = append(enrList, enr)
	}

	var nodes []*enode.Node
	for _, addr := range enrList {
		node, err := enode.Parse(enode.ValidSchemes, addr)
		if err != nil {
			return err
		}
		nodes = append(nodes, node)
	}

	return wakuState.node.DiscV5().SetBootnodes(nodes)
}
