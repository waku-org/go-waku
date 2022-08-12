//go:build !gowaku_rln
// +build !gowaku_rln

package waku

import "github.com/status-im/go-waku/waku/v2/node"

func checkForRLN(options Options, nodeOpts *[]node.WakuNodeOption) {
	// Do nothing
}

func onStartRLN(wakuNode *node.WakuNode, options Options) {
	// Do nothing
}
