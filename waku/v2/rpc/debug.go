package rpc

import (
	"net/http"

	"github.com/status-im/go-waku/waku/v2/node"
)

type DebugService struct {
	node *node.WakuNode
}

type InfoArgs struct {
}

type InfoReply struct {
	ListenAddresses []string `json:"listenAddresses,omitempty"`
}

func (d *DebugService) GetV1Info(r *http.Request, args *InfoArgs, reply *InfoReply) error {
	for _, addr := range d.node.ListenAddresses() {
		reply.ListenAddresses = append(reply.ListenAddresses, addr.String())
	}
	return nil
}

type VersionResponse string

func (d *DebugService) GetV1Version(r *http.Request, args *InfoArgs, reply *VersionResponse) error {
	*reply = VersionResponse(node.GitCommit)
	return nil
}
