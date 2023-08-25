//go:build gowaku_rln
// +build gowaku_rln

package main

import (
	cli "github.com/urfave/cli/v2"
	wcli "github.com/waku-org/go-waku/waku/cliutils"
)

func rlnFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "rln-relay",
			Value:       false,
			Usage:       "Enable spam protection through rln-relay",
			Destination: &options.RLNRelay.Enable,
		},
		&cli.UintFlag{
			Name:        "rln-relay-membership-group-index",
			Value:       0,
			Usage:       "the index of credentials to use, within a specific rln membership set",
			Destination: &options.RLNRelay.MembershipGroupIndex,
		},
		&cli.BoolFlag{
			Name:        "rln-relay-dynamic",
			Usage:       "Enable waku-rln-relay with on-chain dynamic group management",
			Destination: &options.RLNRelay.Dynamic,
		},
		&cli.PathFlag{
			Name:        "rln-relay-cred-path",
			Usage:       "RLN relay membership credentials file",
			Value:       "",
			Destination: &options.RLNRelay.CredentialsPath,
		},
		&cli.StringFlag{
			Name:        "rln-relay-cred-password",
			Value:       "",
			Usage:       "Password for encrypting RLN credentials",
			Destination: &options.RLNRelay.CredentialsPassword,
		},
		&cli.StringFlag{
			Name:        "rln-relay-tree-path",
			Value:       "",
			Usage:       "Path to the RLN merkle tree sled db (https://github.com/spacejam/sled)",
			Destination: &options.RLNRelay.TreePath,
		},
		&cli.UintFlag{
			Name:        "rln-relay-membership-index",
			Value:       0,
			Usage:       "the index of credentials to use",
			Destination: &options.RLNRelay.CredentialsIndex,
		},
		&cli.StringFlag{
			Name:        "rln-relay-eth-client-address",
			Usage:       "Ethereum testnet client address",
			Value:       "ws://localhost:8545",
			Destination: &options.RLNRelay.ETHClientAddress,
		},
		&cli.GenericFlag{
			Name:  "rln-relay-eth-contract-address",
			Usage: "Address of membership contract",
			Value: &wcli.AddressValue{
				Value: &options.RLNRelay.MembershipContractAddress,
			},
		},
	}
}
