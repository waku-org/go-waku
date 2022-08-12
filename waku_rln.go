//go:build gowaku_rln
// +build gowaku_rln

package main

import "github.com/urfave/cli/v2"

func rlnFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "rln-relay",
			Value:       false,
			Usage:       "Enable spam protection through rln-relay",
			Destination: &options.RLNRelay.Enable,
		},
		&cli.IntFlag{
			Name:        "rln-relay-membership-index",
			Value:       0,
			Usage:       "(experimental) the index of node in the rln-relay group: a value between 0-99 inclusive",
			Destination: &options.RLNRelay.MembershipIndex,
		},
		&cli.StringFlag{
			Name:        "rln-relay-pubsub-topic",
			Value:       "/waku/2/default-waku/proto",
			Usage:       "the pubsub topic for which rln-relay gets enabled",
			Destination: &options.RLNRelay.PubsubTopic,
		},
		&cli.StringFlag{
			Name:        "rln-relay-content-topic",
			Value:       "/toy-chat/2/luzhou/proto",
			Usage:       "the content topic for which rln-relay gets enabled",
			Destination: &options.RLNRelay.ContentTopic,
		},
		&cli.BoolFlag{
			Name:        "rln-relay-dynamic",
			Usage:       "Enable waku-rln-relay with on-chain dynamic group management",
			Destination: &options.RLNRelay.Dynamic,
		},
		&cli.StringFlag{
			Name:        "rln-relay-id",
			Usage:       "Rln relay identity secret key as a Hex string",
			Destination: &options.RLNRelay.IDKey,
		},
		&cli.StringFlag{
			Name:        "rln-relay-id-commitment",
			Usage:       "Rln relay identity commitment key as a Hex string",
			Destination: &options.RLNRelay.IDCommitment,
		},
		&cli.StringFlag{
			Name:        "rln-relay-membership-credentials-file",
			Usage:       "RLN relay membership credentials file",
			Value:       "rlnCredentials.txt",
			Destination: &options.RLNRelay.CredentialsFile,
		},
		// TODO: this is a good candidate option for subcommands
		// TODO: consider accepting a private key file and passwd
		&cli.StringFlag{
			Name:        "eth-private-key",
			Usage:       "Ethereum Goerli testnet account private key used for registering in member contract",
			Destination: &options.RLNRelay.ETHPrivateKey,
		},
		&cli.StringFlag{
			Name:        "eth-client-address",
			Usage:       "Ethereum testnet client address",
			Value:       "ws://localhost:8545",
			Destination: &options.RLNRelay.ETHClientAddress,
		},
		&cli.StringFlag{
			Name:        "eth-mem-contract-address",
			Usage:       "Address of membership contract on an Ethereum testnet",
			Destination: &options.RLNRelay.MembershipContractAddress,
		},
	}
}
