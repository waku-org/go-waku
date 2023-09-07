//go:build !gowaku_rln
// +build !gowaku_rln

package server

import (
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

func AppendRLNProof(node *node.WakuNode, msg *pb.WakuMessage) error {
	return nil
}
