package peermanager

import (
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"testing"
)

func TestSubscribeToRelayEvtBus(t *testing.T) {
	log := utils.Logger()

	// Host 1
	r, _, h1, _ := tests.MakeWakuRelay(t, "testTopic", log)

	// Host 1 used by peer manager
	pm := NewPeerManager(10, 20, utils.Logger())
	pm.SetHost(h1)

	// Create a new relay event bus
	relayEvtBus := r.Events()

	// Subscribe to EventBus
	err := pm.SubscribeToRelayEvtBus(relayEvtBus)
	require.NoError(t, err)

}
