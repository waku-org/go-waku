package lightpush

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/status-im/go-waku/tests"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/require"
)

func TestLightPushOption(t *testing.T) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	options := []LightPushOption{
		WithPeer("QmWLxGxG65CZ7vRj5oNXCJvbY9WkF9d9FxuJg8cg8Y7q3"),
		WithAutomaticPeerSelection(host),
		WithFastestPeerSelection(context.Background()),
		WithRequestId([]byte("requestId")),
		WithAutomaticRequestId(),
	}

	params := new(LightPushParameters)
	params.host = host
	params.log = utils.Logger()

	for _, opt := range options {
		opt(params)
	}

	require.Equal(t, host, params.host)
	require.NotNil(t, params.selectedPeer)
	require.NotNil(t, params.requestId)
}
