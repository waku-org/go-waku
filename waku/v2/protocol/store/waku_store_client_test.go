package store

import (
	"context"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"testing"
)

func TestQueryOptions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pubSubTopic := "/waku/2/go/store/test"
	contentTopic := "/test/2/my-app"

	host, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	host2.Peerstore().AddAddr(host.ID(), tests.GetHostAddress(host), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host.ID(), StoreID_v20beta4)
	require.NoError(t, err)

	msg := tests.CreateWakuMessage(contentTopic, utils.GetUnixEpoch())

	s := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s.SetHost(host)
	_ = s.storeMessage(protocol.NewEnvelope(msg, *utils.GetUnixEpoch(), pubSubTopic))

	sub := SimulateSubscription([]*protocol.Envelope{
		protocol.NewEnvelope(msg, *utils.GetUnixEpoch(), pubSubTopic),
	})

	err = s.Start(ctx, sub)
	require.NoError(t, err)
	defer s.Stop()

	s2 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s2.SetHost(host2)

	sub2 := relay.NewSubscription(protocol.NewContentFilter(relay.DefaultWakuTopic))

	err = s2.Start(ctx, sub2)
	require.NoError(t, err)
	defer s2.Stop()

	q := Query{
		PubsubTopic:   pubSubTopic,
		ContentTopics: []string{contentTopic},
		StartTime:     nil,
		EndTime:       nil,
	}

	_, err = s.Query(ctx, q, WithPeer(host.ID()), WithPeerAddr(tests.GetHostAddress(host)))
	require.Error(t, err)

	_, err = s.Query(ctx, q, WithPeerAddr(tests.GetHostAddress(host)), WithPeer(host.ID()))
	require.Error(t, err)

	_, err = s2.Query(ctx, q, WithPeer(host.ID()), WithRequestID([]byte("requestID")))
	require.NoError(t, err)

}
