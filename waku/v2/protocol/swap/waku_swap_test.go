package swap

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestSwapCreditDebit(t *testing.T) {
	swap := NewWakuSwap(utils.Logger(), []SwapOption{
		WithMode(SoftMode),
		WithThreshold(0, 0),
	}...)

	swap.Credit("1", 1)
	require.Equal(t, -1, swap.Accounting["1"])

	swap.Debit("1", 2)
	require.Equal(t, 1, swap.Accounting["1"])
}
