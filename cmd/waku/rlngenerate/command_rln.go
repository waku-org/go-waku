//go:build gowaku_rln
// +build gowaku_rln

package rlngenerate

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
	cli "github.com/urfave/cli/v2"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager/dynamic"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/keystore"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

var options Options
var logger = utils.Logger().Named("rln-credentials")

// Command generates a key file used to generate the node's peerID, encrypted with an optional password
var Command = cli.Command{
	Name:  "generate-rln-credentials",
	Usage: "Generate credentials for usage with RLN",
	Action: func(cCtx *cli.Context) error {
		if options.ETHPrivateKey == nil {
			err := errors.New("a private key must be specified")
			logger.Error("validating option flags", zap.Error(err))
			return cli.Exit(err, 1)
		}

		err := execute(context.Background())
		if err != nil {
			logger.Error("registering RLN credentials", zap.Error(err))
			return cli.Exit(err, 1)
		}

		return nil
	},
	Flags: flags,
}

func execute(ctx context.Context) error {
	ethClient, err := ethclient.Dial(options.ETHClientAddress)
	if err != nil {
		return err
	}

	rlnInstance, err := rln.NewRLN()
	if err != nil {
		return err
	}

	chainID, err := ethClient.ChainID(ctx)
	if err != nil {
		return err
	}

	rlnContract, err := contracts.NewRLN(options.MembershipContractAddress, ethClient)
	if err != nil {
		return err
	}

	// prepare rln membership key pair
	logger.Info("generating rln credential")
	identityCredential, err := rlnInstance.MembershipKeyGen()
	if err != nil {
		return err
	}

	// register the rln-relay peer to the membership contract
	membershipIndex, err := register(ctx, ethClient, rlnContract, identityCredential.IDCommitment, chainID)
	if err != nil {
		return err
	}

	// TODO: clean private key from memory

	err = persistCredentials(identityCredential, membershipIndex, chainID)
	if err != nil {
		return err
	}

	if logger.Level() == zap.DebugLevel {
		logger.Info("registered credentials into the membership contract",
			logging.HexString("IDCommitment", identityCredential.IDCommitment[:]),
			logging.HexString("IDNullifier", identityCredential.IDNullifier[:]),
			logging.HexString("IDSecretHash", identityCredential.IDSecretHash[:]),
			logging.HexString("IDTrapDoor", identityCredential.IDTrapdoor[:]),
			zap.Uint("index", membershipIndex),
		)
	} else {
		logger.Info("registered credentials into the membership contract", logging.HexString("idCommitment", identityCredential.IDCommitment[:]), zap.Uint("index", membershipIndex))
	}

	ethClient.Close()

	return nil
}

func persistCredentials(identityCredential *rln.IdentityCredential, membershipIndex rln.MembershipIndex, chainID *big.Int) error {
	appKeystore, err := keystore.New(options.CredentialsPath, dynamic.RLNAppInfo, logger)
	if err != nil {
		return err
	}

	membershipGroup := keystore.MembershipGroup{
		TreeIndex: membershipIndex,
		MembershipContract: keystore.MembershipContract{
			ChainID: fmt.Sprintf("0x%X", chainID.Int64()),
			Address: options.MembershipContractAddress.String(),
		},
	}

	membershipGroupIndex, err := appKeystore.AddMembershipCredentials(identityCredential, membershipGroup, options.CredentialsPassword)
	if err != nil {
		return fmt.Errorf("failed to persist credentials: %w", err)
	}

	// TODO: obtain keystore index?
	logger.Info("persisted credentials succesfully", zap.Uint("membershipGroupIndex", membershipGroupIndex))

	return nil
}
