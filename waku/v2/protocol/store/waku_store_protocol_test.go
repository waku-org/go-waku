package store

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/require"
)

func getHostAddress(ha host.Host) ma.Multiaddr {
	return ha.Addrs()[0]
}

func TestWakuStoreProtocolQuery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(true, nil)
	s1.Start(ctx, host1)
	defer s1.Stop()

	topic1 := "1"
	pubsubTopic1 := "topic1"

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic1,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}
	host2, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	// Simulate a message has been received via relay protocol
	s1.MsgC <- protocol.NewEnvelope(msg, pubsubTopic1)

	s2 := NewWakuStore(false, nil)
	s2.Start(ctx, host2)
	defer s2.Stop()

	host2.Peerstore().AddAddr(host1.ID(), getHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), string(StoreID_v20beta3))
	require.NoError(t, err)

	response, err := s2.Query(ctx, &pb.HistoryQuery{
		PubsubTopic: pubsubTopic1,
		ContentFilters: []*pb.ContentFilter{{
			ContentTopic: topic1,
		}},
	}, DefaultOptions()...)

	require.NoError(t, err)
	require.Len(t, response.Messages, 1)
	require.Equal(t, msg, response.Messages[0])
}
