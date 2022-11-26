package rpc

import (
	"net/http"

	"github.com/waku-org/go-waku/waku/v2/node"
)

type DebugService struct {
	node *node.WakuNode
}

type InfoArgs struct {
}

type InfoReply struct {
	ENRUri          string   `json:"enrUri,omitempty"`
	ListenAddresses []string `json:"listenAddresses,omitempty"`
}

func NewDebugService(node *node.WakuNode) *DebugService {
	return &DebugService{
		node: node,
	}
}

func (d *DebugService) GetV1Info(r *http.Request, args *InfoArgs, reply *InfoReply) error {
	reply.ENRUri = d.node.ENR().String()
	for _, addr := range d.node.ListenAddresses() {
		reply.ListenAddresses = append(reply.ListenAddresses, addr.String())
	}
	return nil
}

type VersionResponse string

func (d *DebugService) GetV1Version(r *http.Request, args *InfoArgs, reply *VersionResponse) error {
	*reply = VersionResponse(node.GetVersionInfo().String())
	return nil
}
