package dynamic

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/web3"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"github.com/waku-org/go-zerokit-rln/rln"
)

var NULL_ADDR common.Address = common.HexToAddress("0x0000000000000000000000000000000000000000")

func TestFetchingLogic(t *testing.T) {
	client := NewMockClient(t, "web3_test.json")

	rlnContract, err := contracts.NewRLN(NULL_ADDR, client)
	if err != nil {
		t.Fatal(err)
	}
	rlnInstance, err := rln.NewRLN()
	if err != nil {
		t.Fatal(err)
	}

	dgm := DynamicGroupManager{
		web3Config: &web3.Config{
			RLNContract: web3.RLNContract{
				RLN: rlnContract,
			},
			ETHClient: client,
		},
		rln: rlnInstance,
		log: utils.Logger(),
	}

	counts := []int{}
	mockFn := func(_ *DynamicGroupManager, events []*contracts.RLNMemberRegistered) error {
		counts = append(counts, len(events))
		return nil
	}
	// check if more than 10k error is handled or not.
	client.SetErrorOnBlock(5007, fmt.Errorf("query returned more than 10000 results"), 2)
	// loadOldEvents will check till 10010
	client.SetLatestBlockNumber(10010)
	// watchNewEvents will check till 10012
	if err := dgm.HandleGroupUpdates(context.TODO(), mockFn); err != nil {
		t.Fatal(err)
	}
	// sleep so that watchNewEvents can finish
	time.Sleep(time.Second)
	// check whether all the events are fetched or not.
	if !sameArr(counts, []int{1, 3, 2, 1, 1}) {
		t.Fatal("wrong no of events fetched per cycle", counts)
	}
}

func sameArr(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
