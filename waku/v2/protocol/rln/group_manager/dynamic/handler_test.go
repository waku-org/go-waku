package dynamic

import (
	"context"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"github.com/waku-org/go-zerokit-rln/rln"
)

func eventBuilder(blockNumber uint64, removed bool, pubkey int64, index int64) *contracts.RLNMemberRegistered {
	return &contracts.RLNMemberRegistered{
		Raw: types.Log{
			BlockNumber: blockNumber,
			Removed:     removed,
		},
		Index:  big.NewInt(index),
		Pubkey: big.NewInt(pubkey),
	}
}

func TestHandler(t *testing.T) {
	// Create a RLN instance
	rlnInstance, err := rln.NewRLN()
	require.NoError(t, err)

	rootTracker, err := group_manager.NewMerkleRootTracker(5, rlnInstance)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	_ = ctx

	gm := &DynamicGroupManager{
		rln:         rlnInstance,
		log:         utils.Logger(),
		cancel:      cancel,
		wg:          sync.WaitGroup{},
		rootTracker: rootTracker,
	}

	root0 := [32]byte{62, 31, 25, 34, 223, 182, 113, 211, 249, 18, 247, 234, 70, 30, 10, 136, 238, 132, 143, 221, 225, 43, 108, 24, 171, 26, 210, 197, 106, 231, 52, 33}
	roots := gm.rootTracker.Roots()
	require.Len(t, roots, 1)
	require.Equal(t, roots[0], root0)

	events := []*contracts.RLNMemberRegistered{eventBuilder(1, false, 0xaaaa, 1)}

	err = handler(gm, events)
	require.NoError(t, err)

	roots = gm.rootTracker.Roots()

	require.Len(t, roots, 2)
	require.Equal(t, root0, roots[0])
	require.Equal(t, [32]byte{0x96, 0x65, 0xdb, 0xbc, 0x28, 0x6b, 0x1f, 0xc6, 0xab, 0x2d, 0x82, 0xe, 0x30, 0xe0, 0xc1, 0x92, 0x32, 0x73, 0xd2, 0xd6, 0xe1, 0x21, 0xef, 0xa1, 0xc8, 0xa6, 0xa6, 0xd, 0x9, 0x7c, 0x85, 0x19}, roots[1])

	events = []*contracts.RLNMemberRegistered{
		eventBuilder(1, false, 0xbbbb, 2),
		eventBuilder(2, false, 0xcccc, 3),
		eventBuilder(3, false, 0xdddd, 4),
		eventBuilder(4, false, 0xeeee, 5),
	}

	err = handler(gm, events)
	require.NoError(t, err)

	// Root[1] should become [0]
	roots = gm.rootTracker.Roots()
	require.Len(t, roots, 5)
	require.Equal(t, [32]byte{0x96, 0x65, 0xdb, 0xbc, 0x28, 0x6b, 0x1f, 0xc6, 0xab, 0x2d, 0x82, 0xe, 0x30, 0xe0, 0xc1, 0x92, 0x32, 0x73, 0xd2, 0xd6, 0xe1, 0x21, 0xef, 0xa1, 0xc8, 0xa6, 0xa6, 0xd, 0x9, 0x7c, 0x85, 0x19}, roots[0])
	require.Len(t, rootTracker.Buffer(), 1)
	require.Equal(t, root0, rootTracker.Buffer()[0])

	// We detect a fork
	//
	// [0] -> [1] -> [2] -> [3] -> [4]   Our chain
	//                \
	//                 \-->              Real chain
	// We should restore the valid roots from the buffer at the state the moment the chain forked
	// In this case, just adding the original merkle root from empty tree
	validRootsBeforeFork := roots[0:3]
	events = []*contracts.RLNMemberRegistered{eventBuilder(3, true, 0xdddd, 4)}

	err = handler(gm, events)
	require.NoError(t, err)

	roots = gm.rootTracker.Roots()
	require.Len(t, roots, 4)
	require.Equal(t, roots[0], root0)
	require.Equal(t, roots[1], validRootsBeforeFork[0])
	require.Equal(t, roots[2], validRootsBeforeFork[1])
	require.Equal(t, roots[3], validRootsBeforeFork[2])
	require.Len(t, rootTracker.Buffer(), 0)

	// Adding multiple events for same block
	events = []*contracts.RLNMemberRegistered{
		eventBuilder(3, false, 0xdddd, 4),
		eventBuilder(3, false, 0xeeee, 5),
	}

	err = handler(gm, events)
	require.NoError(t, err)

	roots = gm.rootTracker.Roots()
	require.Len(t, roots, 5)

}
