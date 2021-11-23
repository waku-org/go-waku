package swap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSwapOption(t *testing.T) {
	options := []SwapOption{
		WithMode(SoftMode),
		WithThreshold(10, 0),
	}

	params := &SwapParameters{}

	for _, opt := range options {
		opt(params)
	}

	require.Equal(t, SoftMode, params.mode)
	require.Equal(t, 10, params.paymentThreshold)
	require.Equal(t, 0, params.disconnectThreshold)
}
