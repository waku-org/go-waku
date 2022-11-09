//go:build gowaku_rln
// +build gowaku_rln

package waku

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
			membershipCredentials, err := node.GetMembershipCredentials(
				logger,
				options.RLNRelay.CredentialsPath,
				options.RLNRelay.CredentialsPassword,
				options.RLNRelay.MembershipContractAddress,
				uint(options.RLNRelay.MembershipIndex),
			)
			failOnErr(err, "Invalid membership credentials")

			*nodeOpts = append(*nodeOpts, node.WithDynamicRLNRelay(
				options.RLNRelay.PubsubTopic,
				options.RLNRelay.ContentTopic,
				membershipCredentials,
				nil,
				options.RLNRelay.ETHClientAddress,
				ethPrivKey,
				nil,
			))
		}
	}
}

func onStartRLN(wakuNode *node.WakuNode, options Options) {
	if options.RLNRelay.Enable && options.RLNRelay.Dynamic && options.RLNRelay.CredentialsPath != "" {
		err := node.WriteRLNMembershipCredentialsToFile(wakuNode.RLNRelay().MembershipKeyPair(), wakuNode.RLNRelay().MembershipIndex(), wakuNode.RLNRelay().MembershipContractAddress(), options.RLNRelay.CredentialsPath, []byte(options.RLNRelay.CredentialsPassword))
		failOnErr(err, "Could not write membership credentials file")
	}
}
