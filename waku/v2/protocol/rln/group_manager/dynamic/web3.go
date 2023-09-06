package dynamic

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	"go.uber.org/zap"
)

// the types of inputs to this handler matches the MemberRegistered event/proc defined in the MembershipContract interface
type RegistrationEventHandler = func(*DynamicGroupManager, []*contracts.RLNMemberRegistered) error

// HandleGroupUpdates mounts the supplied handler for the registration events emitting from the membership contract
// It connects to the eth client, subscribes to the `MemberRegistered` event emitted from the `MembershipContract`
// and collects all the events, for every received event, it calls the `handler`
func (gm *DynamicGroupManager) HandleGroupUpdates(ctx context.Context, handler RegistrationEventHandler) error {
	fromBlock := gm.web3Config.RLNContract.DeployedBlockNumber
	metadata, err := gm.GetMetadata()
	if err != nil {
		gm.log.Warn("could not load last processed block from metadata. Starting onchain sync from deployment block", zap.Error(err), zap.Uint64("deploymentBlock", gm.web3Config.RLNContract.DeployedBlockNumber))
	} else {
		if gm.web3Config.ChainID.Cmp(metadata.ChainID) != 0 {
			return errors.New("persisted data: chain id mismatch")
		}

		if !bytes.Equal(gm.web3Config.RegistryContract.Address.Bytes(), metadata.ContractAddress.Bytes()) {
			return errors.New("persisted data: contract address mismatch")
		}

		fromBlock = metadata.LastProcessedBlock + 1
		gm.log.Info("resuming onchain sync", zap.Uint64("fromBlock", fromBlock))
	}

	gm.rootTracker.SetValidRootsPerBlock(metadata.ValidRootsPerBlock)
	//
	latestBlockNumber, err := gm.latestBlockNumber(ctx)
	if err != nil {
		return err
	}
	//
	err = gm.loadOldEvents(ctx, gm.web3Config.RLNContract.RLN, fromBlock, latestBlockNumber, handler)
	if err != nil {
		return err
	}

	errCh := make(chan error)

	gm.wg.Add(1)
	go gm.watchNewEvents(ctx, latestBlockNumber+1, handler, errCh) // we have already fetched the events for latestBlocNumber in oldEvents
	return <-errCh
}

func (gm *DynamicGroupManager) loadOldEvents(ctx context.Context, rlnContract *contracts.RLN, fromBlock, toBlock uint64, handler RegistrationEventHandler) error {
	for ; fromBlock+maxBatchSize < toBlock; fromBlock += maxBatchSize + 1 { // check if the end of the batch is within the toBlock range
		events, err := gm.getEvents(ctx, fromBlock, fromBlock+maxBatchSize)
		if err != nil {
			return err
		}
		if err := handler(gm, events); err != nil {
			return err
		}
	}

	//
	events, err := gm.getEvents(ctx, fromBlock, toBlock)
	if err != nil {
		return err
	}
	// process all the fetched events
	return handler(gm, events)
}

func (gm *DynamicGroupManager) watchNewEvents(ctx context.Context, fromBlock uint64, handler RegistrationEventHandler, errCh chan<- error) {
	defer gm.wg.Done()

	// Watch for new events
	firstErr := true
	headerCh := make(chan *types.Header)
	subs := event.Resubscribe(2*time.Second, func(ctx context.Context) (event.Subscription, error) {
		s, err := gm.web3Config.ETHClient.SubscribeNewHead(ctx, headerCh)
		if err != nil {
			if err == rpc.ErrNotificationsUnsupported {
				err = errors.New("notifications not supported. The node must support websockets")
			}
			gm.log.Error("subscribing to rln events", zap.Error(err))
		}
		if firstErr { // errCh can be closed only once
			errCh <- err
			close(errCh)
			firstErr = false
		}
		return s, err
	})

	defer subs.Unsubscribe()
	defer close(headerCh)

	for {
		select {
		case h := <-headerCh:
			toBlock := h.Number.Uint64()
			events, err := gm.getEvents(ctx, fromBlock, toBlock)
			if err != nil {
				gm.log.Error("obtaining rln events", zap.Error(err))
			}
			// update the last processed block
			fromBlock = toBlock + 1

			err = handler(gm, events)
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

const maxBatchSize = uint64(5000)

func tooMuchDataRequestedError(err error) bool {
	// this error is only infura specific (other providers might have different error messages)
	return err.Error() == "query returned more than 10000 results"
}

func (gm *DynamicGroupManager) latestBlockNumber(ctx context.Context) (uint64, error) {
	block, err := gm.web3Config.ETHClient.BlockByNumber(ctx, nil)
	if err != nil {
		return 0, err
	}

	return block.Number().Uint64(), nil
}

func (gm *DynamicGroupManager) getEvents(ctx context.Context, fromBlock uint64, toBlock uint64) ([]*contracts.RLNMemberRegistered, error) {
	evts, err := gm.fetchEvents(ctx, fromBlock, toBlock)
	if err != nil {
		if tooMuchDataRequestedError(err) { // divide the range and try again
			mid := (fromBlock + toBlock) / 2
			firstHalfEvents, err := gm.getEvents(ctx, fromBlock, mid)
			if err != nil {
				return nil, err
			}
			secondHalfEvents, err := gm.getEvents(ctx, mid+1, toBlock)
			if err != nil {
				return nil, err
			}
			return append(firstHalfEvents, secondHalfEvents...), nil
		}
		return nil, err
	}
	return evts, nil
}

func (gm *DynamicGroupManager) fetchEvents(ctx context.Context, from uint64, to uint64) ([]*contracts.RLNMemberRegistered, error) {
	logIterator, err := gm.web3Config.RLNContract.FilterMemberRegistered(&bind.FilterOpts{Start: from, End: &to, Context: ctx})
	if err != nil {
		return nil, err
	}

	var results []*contracts.RLNMemberRegistered

	for {
		if !logIterator.Next() {
			break
		}

		if logIterator.Error() != nil {
			return nil, logIterator.Error()
		}

		results = append(results, logIterator.Event)
	}

	return results, nil
}
