package dynamic

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager"
	"github.com/waku-org/go-zerokit-rln/rln"
	r "github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

type DynamicGroupManager struct {
	rln *rln.RLN
	log *zap.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup

	identityCredential        *rln.IdentityCredential
	membershipIndex           *rln.MembershipIndex
	membershipContractAddress common.Address
	ethClientAddress          string
	ethClient                 *ethclient.Client

	// ethAccountPrivateKey is required for signing transactions
	// TODO may need to erase this ethAccountPrivateKey when is not used
	// TODO may need to make ethAccountPrivateKey mandatory
	ethAccountPrivateKey *ecdsa.PrivateKey

	registrationHandler RegistrationHandler
	lastIndexLoaded     int64

	rootTracker *group_manager.MerkleRootTracker
}

type RegistrationHandler = func(tx *types.Transaction)

func NewDynamicGroupManager(
	ethClientAddr string,
	ethAccountPrivateKey *ecdsa.PrivateKey,
	memContractAddr common.Address,
	identityCredential *rln.IdentityCredential,
	index rln.MembershipIndex,
	registrationHandler RegistrationHandler,
	log *zap.Logger,
) (*DynamicGroupManager, error) {
	return &DynamicGroupManager{
		identityCredential:        identityCredential,
		membershipIndex:           &index,
		membershipContractAddress: memContractAddr,
		ethClientAddress:          ethClientAddr,
		ethAccountPrivateKey:      ethAccountPrivateKey,
		registrationHandler:       registrationHandler,
		lastIndexLoaded:           -1,
	}, nil
}

func (gm *DynamicGroupManager) Start(ctx context.Context, rlnInstance *rln.RLN, rootTracker *group_manager.MerkleRootTracker) error {
	if gm.cancel != nil {
		return errors.New("already started")
	}

	ctx, cancel := context.WithCancel(ctx)
	gm.cancel = cancel

	gm.log.Info("mounting rln-relay in on-chain/dynamic mode")

	gm.rln = rlnInstance
	gm.rootTracker = rootTracker

	err := rootTracker.Sync()
	if err != nil {
		return err
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
		membershipIndex, err := gm.Register(ctx)
		if err != nil {
			return err
		}

		gm.membershipIndex = membershipIndex

		gm.log.Info("registered peer into the membership contract")
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
