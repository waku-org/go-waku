package dynamic

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/keystore"
	"github.com/waku-org/go-zerokit-rln/rln"
	r "github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

var RLNAppInfo = keystore.AppInfo{
	Application:   "go-waku-rln-relay",
	AppIdentifier: "01234567890abcdef",
	Version:       "0.1",
}

type DynamicGroupManager struct {
	rln *rln.RLN
	log *zap.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup

	identityCredential *rln.IdentityCredential
	membershipIndex    *rln.MembershipIndex

	membershipContractAddress common.Address
	ethClientAddress          string
	ethClient                 *ethclient.Client

	// ethAccountPrivateKey is required for signing transactions
	// TODO may need to erase this ethAccountPrivateKey when is not used
	// TODO may need to make ethAccountPrivateKey mandatory
	ethAccountPrivateKey *ecdsa.PrivateKey

	registrationHandler RegistrationHandler
	chainId             *big.Int
	rlnContract         *contracts.RLN
	membershipFee       *big.Int
	lastIndexLoaded     int64

	saveKeystore     bool
	keystorePath     string
	keystorePassword string

	rootTracker *group_manager.MerkleRootTracker
}

type RegistrationHandler = func(tx *types.Transaction)

func NewDynamicGroupManager(
	ethClientAddr string,
	ethAccountPrivateKey *ecdsa.PrivateKey,
	memContractAddr common.Address,
	keystorePath string,
	keystorePassword string,
	saveKeystore bool,
	registrationHandler RegistrationHandler,
	log *zap.Logger,
) (*DynamicGroupManager, error) {
	return &DynamicGroupManager{
		membershipContractAddress: memContractAddr,
		ethClientAddress:          ethClientAddr,
		ethAccountPrivateKey:      ethAccountPrivateKey,
		registrationHandler:       registrationHandler,
		lastIndexLoaded:           -1,
		saveKeystore:              saveKeystore,
		keystorePath:              keystorePath,
		keystorePassword:          keystorePassword,
	}, nil
}

func (gm *DynamicGroupManager) getMembershipFee(ctx context.Context) (*big.Int, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(gm.ethAccountPrivateKey, gm.chainId)
	if err != nil {
		return nil, err
	}
	auth.Context = ctx

	return gm.rlnContract.MEMBERSHIPDEPOSIT(&bind.CallOpts{Context: ctx})
}

func (gm *DynamicGroupManager) Start(ctx context.Context, rlnInstance *rln.RLN, rootTracker *group_manager.MerkleRootTracker) error {
	if gm.cancel != nil {
		return errors.New("already started")
	}

	ctx, cancel := context.WithCancel(ctx)
	gm.cancel = cancel

	gm.log.Info("mounting rln-relay in on-chain/dynamic mode")

	backend, err := ethclient.Dial(gm.ethClientAddress)
	if err != nil {
		return err
	}
	gm.ethClient = backend

	gm.rln = rlnInstance
	gm.rootTracker = rootTracker

	gm.chainId, err = backend.ChainID(ctx)
	if err != nil {
		return err
	}

	gm.rlnContract, err = contracts.NewRLN(gm.membershipContractAddress, backend)
	if err != nil {
		return err
	}

	// check if the contract exists by calling a static function
	gm.membershipFee, err = gm.getMembershipFee(ctx)
	if err != nil {
		return err
	}

	err = rootTracker.Sync()
	if err != nil {
		return err
	}

	if gm.keystorePassword != "" && gm.keystorePath != "" {
		credentials, err := keystore.GetMembershipCredentials(gm.log,
			gm.keystorePath,
			gm.keystorePassword,
			RLNAppInfo,
			nil,
			[]keystore.MembershipContract{{
				ChainId: gm.chainId.String(),
				Address: gm.membershipContractAddress.Hex(),
			}})
		if err != nil {
			return err
		}

		// TODO: accept an index from the config
		gm.identityCredential = &credentials[0].IdentityCredential
		gm.membershipIndex = &credentials[0].MembershipGroups[0].TreeIndex
	}

	// prepare rln membership key pair
	if gm.identityCredential == nil && gm.ethAccountPrivateKey != nil {
		gm.log.Debug("no rln-relay key is provided, generating one")
		identityCredential, err := rlnInstance.MembershipKeyGen()
		if err != nil {
			return err
		}

		gm.identityCredential = identityCredential

		// register the rln-relay peer to the membership contract
		gm.membershipIndex, err = gm.Register(ctx)
		if err != nil {
			return err
		}

		err = gm.persistCredentials()
		if err != nil {
			return err
		}

		gm.log.Info("registered peer into the membership contract")
	}

	if gm.identityCredential == nil || gm.membershipIndex == nil {
		return errors.New("no credentials available")
	}

	handler := func(pubkey r.IDCommitment, index r.MembershipIndex) error {
		return gm.InsertMember(pubkey)
	}

	errChan := make(chan error)
	gm.wg.Add(1)
	go gm.HandleGroupUpdates(ctx, handler, errChan)
	err = <-errChan
	if err != nil {
		return err
	}

	return nil
}

func (gm *DynamicGroupManager) persistCredentials() error {
	if !gm.saveKeystore {
		return nil
	}

	if gm.identityCredential == nil || gm.membershipIndex == nil {
		return errors.New("no credentials to persist")
	}

	path := gm.keystorePath
	if path == "" {
		gm.log.Warn("keystore: no credentials path set, using default path", zap.String("path", keystore.RLN_CREDENTIALS_FILENAME))
		path = keystore.RLN_CREDENTIALS_FILENAME
	}

	password := gm.keystorePassword
	if password == "" {
		gm.log.Warn("keystore: no credentials password set, using default password", zap.String("password", keystore.RLN_CREDENTIALS_PASSWORD))
		password = keystore.RLN_CREDENTIALS_PASSWORD
	}

	keystoreCred := keystore.MembershipCredentials{
		IdentityCredential: *gm.identityCredential,
		MembershipGroups: []keystore.MembershipGroup{{
			TreeIndex: *gm.membershipIndex,
			MembershipContract: keystore.MembershipContract{
				ChainId: gm.chainId.String(),
				Address: gm.membershipContractAddress.String(),
			},
		}},
	}

	err := keystore.AddMembershipCredentials(path, []keystore.MembershipCredentials{keystoreCred}, password, RLNAppInfo, keystore.DefaultSeparator)
	if err != nil {
		return errors.New("failed to persist credentials")
	}

	return nil
}

func (gm *DynamicGroupManager) InsertMember(pubkey rln.IDCommitment) error {
	gm.log.Debug("a new key is added", zap.Binary("pubkey", pubkey[:]))
	// assuming all the members arrive in order
	err := gm.rln.InsertMember(pubkey)
	if err != nil {
		gm.log.Error("inserting member into merkletree", zap.Error(err))
		return err
	}

	err = gm.rootTracker.Sync()
	if err != nil {
		return err
	}

	return nil
}

func (gm *DynamicGroupManager) IdentityCredentials() (rln.IdentityCredential, error) {
	if gm.identityCredential == nil {
		return rln.IdentityCredential{}, errors.New("identity credential has not been setup")
	}

	return *gm.identityCredential, nil
}

func (gm *DynamicGroupManager) MembershipIndex() (rln.MembershipIndex, error) {
	if gm.membershipIndex == nil {
		return 0, errors.New("membership index has not been setup")
	}

	return *gm.membershipIndex, nil
}

func (gm *DynamicGroupManager) Stop() {
	if gm.cancel == nil {
		return
	}

	gm.cancel()
	gm.wg.Wait()
}
