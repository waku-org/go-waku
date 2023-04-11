package legacy_filter

import (
	"testing"

	"github.com/stretchr/testify/require"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/timesource"
)

func TestFilterMap(t *testing.T) {
	fmap := NewFilterMap(v2.NewBroadcaster(100), timesource.NewDefaultClock())

	filter := Filter{
		PeerID:         "id",
		Topic:          "test",
		ContentFilters: []string{"test"},
		Chan:           make(chan *protocol.Envelope),
	}

	fmap.Set("test", filter)
	res := <-fmap.Items()
	require.Equal(t, "test", res.Key)

	item, ok := fmap.Get("test")
	require.True(t, ok)
	require.Equal(t, "test", item.Topic)

	fmap.Delete("test")

	_, ok = fmap.Get("test")
	require.False(t, ok)
}
