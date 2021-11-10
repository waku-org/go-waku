package relay

import (
	"testing"

	waku_proto "github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/stretchr/testify/require"
)

func TestSubscription(t *testing.T) {
	e := Subscription{
		closed: false,
		C:      make(chan *waku_proto.Envelope, 10),
		quit:   make(chan struct{}),
	}
	e.Unsubscribe()
	require.True(t, e.closed)
}
