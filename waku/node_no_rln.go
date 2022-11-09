//go:build !gowaku_rln
// +build !gowaku_rln

package waku

import (
	"github.com/waku-org/go-waku/waku/v2/node"
	"go.uber.org/zap"
)

func checkForRLN(logger *zap.Logger, options Options, nodeOpts *[]node.WakuNodeOption) {
	// Do nothing
}

func onStartRLN(wakuNode *node.WakuNode, options Options) {
	// Do nothing
}
