//go:build gowaku_rln
// +build gowaku_rln

package main

import (
	"crypto/ecdsa"
	"errors"

	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

func checkForRLN(logger *zap.Logger, options Options, nodeOpts *[]node.WakuNodeOption) {
	if options.RLNRelay.Enable {
		if !options.Relay.Enable {
			failOnErr(errors.New("relay not available"), "Could not enable RLN Relay")
		}
		if !options.RLNRelay.Dynamic {
			*nodeOpts = append(*nodeOpts, node.WithStaticRLNRelay(options.RLNRelay.PubsubTopic, options.RLNRelay.ContentTopic, rln.MembershipIndex(options.RLNRelay.MembershipIndex), nil))
		} else {

			var ethPrivKey *ecdsa.PrivateKey
			if options.RLNRelay.ETHPrivateKey != nil {
				ethPrivKey = options.RLNRelay.ETHPrivateKey
			}

			*nodeOpts = append(*nodeOpts, node.WithDynamicRLNRelay(
				options.RLNRelay.PubsubTopic,
				options.RLNRelay.ContentTopic,
				options.RLNRelay.CredentialsPath,
				options.RLNRelay.CredentialsPassword,
				options.RLNRelay.MembershipContractAddress,
				nil,
				options.RLNRelay.ETHClientAddress,
				ethPrivKey,
				nil,
			))
		}
	}
}
