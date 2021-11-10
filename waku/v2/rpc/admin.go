package rpc

import (
	"net/http"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/status-im/go-waku/waku/v2/node"
)

type AdminService struct {
	node *node.WakuNode
}

type GetPeersArgs struct {
}

type PeersArgs struct {
	Peers []string `json:"peers,omitempty"`
}

type PeerReply struct {
	Multiaddr string `json:"mutliaddr,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	Connected bool   `json:"connected,omitempty"`
}

type PeersReply struct {
	Peers []PeerReply `json:"peers,omitempty"`
}

func (a *AdminService) PostV1Peers(req *http.Request, args *PeersArgs, reply *SuccessReply) error {
	for _, peer := range args.Peers {
		addr, err := ma.NewMultiaddr(peer)
		if err != nil {
			log.Error("Error building multiaddr", err)
			reply.Success = false
			reply.Error = err.Error()
			return nil
		}

		err = a.node.DialPeerWithMultiAddress(req.Context(), addr)
		if err != nil {
			log.Error("Error dialing peers", err)
			reply.Success = false
			reply.Error = err.Error()
			return nil
		}
	}

	reply.Success = true
	return nil
}

func (a *AdminService) GetV1Peers(req *http.Request, args *GetPeersArgs, reply *PeersReply) error {
	peers, err := a.node.Peers()
	if err != nil {
		log.Error("Error getting peers", err)
		return nil
	}
	for _, peer := range peers {
		for idx, addr := range peer.Addrs {
			reply.Peers = append(reply.Peers, PeerReply{
				Multiaddr: addr.String(),
				Protocol:  peer.Protocols[idx],
				Connected: peer.Connected,
			})
		}
	}
	return nil
}
