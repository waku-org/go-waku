package filter

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestFilterOption(t *testing.T) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	// subscribe options
	options := []FilterSubscribeOption{
		WithPeer("QmWLxGxG65CZ7vRj5oNXCJvbY9WkF9d9FxuJg8cg8Y7q3"),
		WithAutomaticPeerSelection(),
		WithFastestPeerSelection(context.Background()),
	}

	params := new(FilterSubscribeParameters)
	params.host = host
	params.log = utils.Logger()

	for _, opt := range options {
		opt(params)
	}

	require.Equal(t, host, params.host)
	require.NotNil(t, params.selectedPeer)

	// Unsubscribe options
	options2 := []FilterSubscribeOption{
		WithAutomaticRequestID(),
		UnsubscribeAll(),
		WithPeer("QmWLxGxG65CZ7vRj5oNXCJvbY9WkF9d9FxuJg8cg8Y7q3"),
	}

	params2 := new(FilterSubscribeParameters)

	for _, opt := range options2 {
		opt(params2)
	}

	require.NotNil(t, params2.selectedPeer)
	require.True(t, params2.unsubscribeAll)

}
