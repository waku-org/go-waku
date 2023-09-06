//go:build gowaku_rln
// +build gowaku_rln

package node

import (
	"bytes"
	"context"
	"errors"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager/dynamic"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager/static"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/keystore"
	r "github.com/waku-org/go-zerokit-rln/rln"
)

// RLNRelay is used to access any operation related to Waku RLN protocol
func (w *WakuNode) RLNRelay() RLNRelay {
	return w.rlnRelay
}

func (w *WakuNode) setupRLNRelay() error {
	var err error

	if !w.opts.enableRLN {
		return nil
	}

	var groupManager group_manager.GroupManager

	rlnInstance, rootTracker, err := rln.GetRLNInstanceAndRootTracker(w.opts.rlnTreePath)
	if err != nil {
		return err
	}
	if !w.opts.rlnRelayDynamic {
		w.log.Info("setting up waku-rln-relay in off-chain mode")

		// set up rln relay inputs
		groupKeys, idCredential, err := static.Setup(w.opts.rlnRelayMemIndex)
		if err != nil {
			return err
		}

		groupManager, err = static.NewStaticGroupManager(groupKeys, idCredential, w.opts.rlnRelayMemIndex, rlnInstance,
			rootTracker, w.log)
		if err != nil {
			return err
		}
	} else {
		w.log.Info("setting up waku-rln-relay in on-chain mode")

		appKeystore, err := keystore.New(w.opts.keystorePath, dynamic.RLNAppInfo, w.log)
		if err != nil {
			return err
		}

		groupManager, err = dynamic.NewDynamicGroupManager(
			w.opts.rlnETHClientAddress,
			w.opts.rlnMembershipContractAddress,
			w.opts.rlnRelayMemIndex,
			appKeystore,
			w.opts.keystorePassword,
			w.opts.prometheusReg,
			rlnInstance,
			rootTracker,
			w.log,
		)
		if err != nil {
			return err
		}
	}

	rlnRelay := rln.New(group_manager.Details{
		GroupManager: groupManager,
		RootTracker:  rootTracker,
		RLN:          rlnInstance,
	}, w.timesource, w.opts.prometheusReg, w.log)

	w.rlnRelay = rlnRelay

	// Adding RLN as a default validator
	w.opts.pubsubOpts = append(w.opts.pubsubOpts, pubsub.WithDefaultValidator(rlnRelay.Validator(w.opts.rlnSpamHandler)))

	return nil
}

func (w *WakuNode) startRlnRelay(ctx context.Context) error {
	rlnRelay := w.rlnRelay.(*rln.WakuRLNRelay)

	err := rlnRelay.Start(ctx)
	if err != nil {
		return err
	}

	if !w.opts.rlnRelayDynamic {
		// check the correct construction of the tree by comparing the calculated root against the expected root
		// no error should happen as it is already captured in the unit tests
		root, err := rlnRelay.RLN.GetMerkleRoot()
		if err != nil {
			return err
		}

		expectedRoot, err := r.ToBytes32LE(r.STATIC_GROUP_MERKLE_ROOT)
		if err != nil {
			return err
		}

		if !bytes.Equal(expectedRoot[:], root[:]) {
			return errors.New("root mismatch: something went wrong not in Merkle tree construction")
		}
	}

	w.log.Info("mounted waku RLN relay")

	return nil
}

func (w *WakuNode) stopRlnRelay() error {
	if w.rlnRelay != nil {
		return w.rlnRelay.Stop()
	}
	return nil
}
