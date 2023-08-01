//go:build !gowaku_rln
// +build !gowaku_rln

package main

import (
	"github.com/waku-org/go-waku/waku/v2/node"
	"go.uber.org/zap"
)

func checkForRLN(logger *zap.Logger, options Options, nodeOpts *[]node.WakuNodeOption) {
	// Do nothing
}
