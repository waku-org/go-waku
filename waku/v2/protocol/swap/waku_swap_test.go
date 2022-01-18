package swap

import (
	"testing"

	logging "github.com/ipfs/go-log"
	"github.com/stretchr/testify/require"
)

var log = logging.Logger("test")

func TestSwapCreditDebit(t *testing.T) {
	swap := NewWakuSwap(&log.SugaredLogger, []SwapOption{
		WithMode(SoftMode),
		WithThreshold(0, 0),
	}...)

	swap.Credit("1", 1)
	require.Equal(t, -1, swap.Accounting["1"])

	swap.Debit("1", 2)
	require.Equal(t, 1, swap.Accounting["1"])
}
