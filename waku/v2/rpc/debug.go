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
	Version string `json:"version,omitempty"`
}

func (d *DebugService) GetV1Info(r *http.Request, args *InfoArgs, reply *InfoReply) error {
	reply.Version = "2.0"
	return nil
}
