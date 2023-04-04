package dynamic

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	r "github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

var MEMBERSHIP_FEE = big.NewInt(1000000000000000) // wei - 0.001 eth

func toBigInt(i []byte) *big.Int {
	result := new(big.Int)
	result.SetBytes(i[:])
	return result
}

func register(ctx context.Context, idComm r.IDCommitment, ethAccountPrivateKey *ecdsa.PrivateKey, ethClientAddress string, membershipContractAddress common.Address, registrationHandler RegistrationHandler, log *zap.Logger) (*r.MembershipIndex, error) {
	backend, err := ethclient.Dial(ethClientAddress)
	if err != nil {
		return nil, err
	}
	defer backend.Close()

	chainID, err := backend.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(ethAccountPrivateKey, chainID)
	if err != nil {
		return nil, err
	}
	auth.Value = MEMBERSHIP_FEE
	auth.Context = ctx

	rlnContract, err := contracts.NewRLN(membershipContractAddress, backend)
	if err != nil {
		return nil, err
	}

	log.Debug("registering an id commitment", zap.Binary("idComm", idComm[:]))

	// registers the idComm  into the membership contract whose address is in rlnPeer.membershipContractAddress
	tx, err := rlnContract.Register(auth, toBigInt(idComm[:]))
	if err != nil {
		return nil, err
	}

	log.Info("transaction broadcasted", zap.String("transactionHash", tx.Hash().Hex()))

	if registrationHandler != nil {
		registrationHandler(tx)
	}

	txReceipt, err := bind.WaitMined(ctx, backend, tx)
	if err != nil {
		return nil, err
	}

	if txReceipt.Status != types.ReceiptStatusSuccessful {
		return nil, errors.New("transaction reverted")
	}

	// the receipt topic holds the hash of signature of the raised events
	evt, err := rlnContract.ParseMemberRegistered(*txReceipt.Logs[0])
	if err != nil {
		return nil, err
	}

	var eventIdComm r.IDCommitment = r.Bytes32(evt.Pubkey.Bytes())

	log.Debug("the identity commitment key extracted from tx log", zap.Binary("eventIdComm", eventIdComm[:]))

	if eventIdComm != idComm {
		return nil, errors.New("invalid id commitment key")
	}

	result := new(r.MembershipIndex)
	*result = r.MembershipIndex(uint(evt.Index.Int64()))

	// debug "the index of registered identity commitment key", eventIndex=eventIndex

	log.Debug("the index of registered identity commitment key", zap.Uint("eventIndex", uint(*result)))

	return result, nil
}

// Register registers the public key of the rlnPeer which is rlnPeer.membershipKeyPair.publicKey
// into the membership contract whose address is in rlnPeer.membershipContractAddress
func (gm *DynamicGroupManager) Register(ctx context.Context) (*r.MembershipIndex, error) {
	pk := gm.identityCredential.IDCommitment
	return register(ctx, pk, gm.ethAccountPrivateKey, gm.ethClientAddress, gm.membershipContractAddress, gm.registrationHandler, gm.log)
}

// the types of inputs to this handler matches the MemberRegistered event/proc defined in the MembershipContract interface
type RegistrationEventHandler = func(pubkey r.IDCommitment, index r.MembershipIndex) error

func (gm *DynamicGroupManager) processLogs(evt *contracts.RLNMemberRegistered, handler RegistrationEventHandler) error {
	if evt == nil {
		return nil
	}

	var pubkey r.IDCommitment = r.Bytes32(evt.Pubkey.Bytes())

	index := evt.Index.Int64()
	if index <= gm.lastIndexLoaded {
		return nil
	}

	gm.lastIndexLoaded = index
	return handler(pubkey, r.MembershipIndex(uint(evt.Index.Int64())))
}

// HandleGroupUpdates mounts the supplied handler for the registration events emitting from the membership contract
// It connects to the eth client, subscribes to the `MemberRegistered` event emitted from the `MembershipContract`
// and collects all the events, for every received event, it calls the `handler`
func (gm *DynamicGroupManager) HandleGroupUpdates(ctx context.Context, handler RegistrationEventHandler) error {
	defer gm.wg.Done()

	backend, err := ethclient.Dial(gm.ethClientAddress)
	if err != nil {
		return err
	}
	gm.ethClient = backend

	rlnContract, err := contracts.NewRLN(gm.membershipContractAddress, backend)
	if err != nil {
		return err
	}

	err = gm.loadOldEvents(ctx, rlnContract, handler)
	if err != nil {
		return err
	}

	errCh := make(chan error)
	go gm.watchNewEvents(ctx, rlnContract, handler, gm.log, errCh)
	return <-errCh
}

func (gm *DynamicGroupManager) loadOldEvents(ctx context.Context, rlnContract *contracts.RLN, handler RegistrationEventHandler) error {
	logIterator, err := rlnContract.FilterMemberRegistered(&bind.FilterOpts{Start: 0, End: nil, Context: ctx})
	if err != nil {
		return err
	}
	for {
		if !logIterator.Next() {
			break
		}

		if logIterator.Error() != nil {
			return logIterator.Error()
		}

		err = gm.processLogs(logIterator.Event, handler)
		if err != nil {
			return err
		}
	}
	return nil
}

func (gm *DynamicGroupManager) watchNewEvents(ctx context.Context, rlnContract *contracts.RLN, handler RegistrationEventHandler, log *zap.Logger, errCh chan<- error) {
	// Watch for new events
	logSink := make(chan *contracts.RLNMemberRegistered)

	firstErr := true
	subs := event.Resubscribe(2*time.Second, func(ctx context.Context) (event.Subscription, error) {
		subs, err := rlnContract.WatchMemberRegistered(&bind.WatchOpts{Context: ctx, Start: nil}, logSink)
		if err != nil {
			if err == rpc.ErrNotificationsUnsupported {
				err = errors.New("notifications not supported. The node must support websockets")
			}
			if firstErr {
				errCh <- err
			}
			gm.log.Error("subscribing to rln events", zap.Error(err))
		}
		firstErr = false
		close(errCh)
		return subs, err
	})

	defer subs.Unsubscribe()
	defer close(logSink)

	for {
		select {
		case evt := <-logSink:
			err := gm.processLogs(evt, handler)
			if err != nil {
				gm.log.Error("processing rln log", zap.Error(err))
			}
		case <-ctx.Done():
			return
		case err := <-subs.Err():
			if err != nil {
				gm.log.Error("watching new events", zap.Error(err))
			}
			return
		}
	}
}
