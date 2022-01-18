package swap

import (
	"testing"

	"github.com/status-im/go-waku/tests"
	"github.com/stretchr/testify/require"
)

func TestSwapCreditDebit(t *testing.T) {
	swap := NewWakuSwap(tests.Logger(), []SwapOption{
		WithMode(SoftMode),
		WithThreshold(0, 0),
	}...)

	swap.Credit("1", 1)
	require.Equal(t, -1, swap.Accounting["1"])

	swap.Debit("1", 2)
	require.Equal(t, 1, swap.Accounting["1"])
}
