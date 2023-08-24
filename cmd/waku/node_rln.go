//go:build gowaku_rln
// +build gowaku_rln

package main

import (
	"errors"

	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

func checkForRLN(logger *zap.Logger, options NodeOptions, nodeOpts *[]node.WakuNodeOption) {
	if options.RLNRelay.Enable {
		if !options.Relay.Enable {
			failOnErr(errors.New("relay not available"), "Could not enable RLN Relay")
		}
		if !options.RLNRelay.Dynamic {
			*nodeOpts = append(*nodeOpts, node.WithStaticRLNRelay(rln.MembershipIndex(options.RLNRelay.MembershipIndex), nil))
		} else {
			// TODO: too many parameters in this function
			// consider passing a config struct instead
			*nodeOpts = append(*nodeOpts, node.WithDynamicRLNRelay(
				options.RLNRelay.CredentialsPath,
				options.RLNRelay.CredentialsPassword,
				options.RLNRelay.TreePath,
				options.RLNRelay.MembershipContractAddress,
				rln.MembershipIndex(options.RLNRelay.MembershipIndex),
				nil,
				options.RLNRelay.ETHClientAddress,
			))
		}
	}
}
