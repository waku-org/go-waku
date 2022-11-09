package store

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestWakuStoreProtocolQuery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(host1, nil, MemoryDB(t), utils.Logger())
	s1.Start(ctx)
	defer s1.Stop()

	topic1 := "1"
	pubsubTopic1 := "topic1"

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic1,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}
	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	// Simulate a message has been received via relay protocol
	s1.MsgC <- protocol.NewEnvelope(msg, utils.GetUnixEpoch(), pubsubTopic1)

	s2 := NewWakuStore(host2, nil, MemoryDB(t), utils.Logger())
	s2.Start(ctx)
	defer s2.Stop()

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), string(StoreID_v20beta4))
	require.NoError(t, err)

	q := Query{
		Topic:         pubsubTopic1,
		ContentTopics: []string{topic1},
	}

	response, err := s2.Query(ctx, q, DefaultOptions()...)

	require.NoError(t, err)
	require.Len(t, response.Messages, 1)
	require.Equal(t, msg, response.Messages[0])
}

func TestWakuStoreProtocolNext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	db := MemoryDB(t)

	s1 := NewWakuStore(host1, nil, db, utils.Logger())
	s1.Start(ctx)
	defer s1.Stop()

	topic1 := "1"
	pubsubTopic1 := "topic1"

	msg1 := tests.CreateWakuMessage(topic1, 1)
	msg2 := tests.CreateWakuMessage(topic1, 2)
	msg3 := tests.CreateWakuMessage(topic1, 3)
	msg4 := tests.CreateWakuMessage(topic1, 4)
	msg5 := tests.CreateWakuMessage(topic1, 5)

	s1.MsgC <- protocol.NewEnvelope(msg1, utils.GetUnixEpoch(), pubsubTopic1)
	s1.MsgC <- protocol.NewEnvelope(msg2, utils.GetUnixEpoch(), pubsubTopic1)
	s1.MsgC <- protocol.NewEnvelope(msg3, utils.GetUnixEpoch(), pubsubTopic1)
	s1.MsgC <- protocol.NewEnvelope(msg4, utils.GetUnixEpoch(), pubsubTopic1)
	s1.MsgC <- protocol.NewEnvelope(msg5, utils.GetUnixEpoch(), pubsubTopic1)

	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), string(StoreID_v20beta4))
	require.NoError(t, err)

	s2 := NewWakuStore(host2, nil, db, utils.Logger())
	s2.Start(ctx)
	defer s2.Stop()

	q := Query{
		Topic:         pubsubTopic1,
		ContentTopics: []string{topic1},
	}

	response, err := s2.Query(ctx, q, WithAutomaticPeerSelection(), WithAutomaticRequestId(), WithPaging(true, 2))
	require.NoError(t, err)
	require.Len(t, response.Messages, 2)
	require.Equal(t, response.Messages[0].Timestamp, msg1.Timestamp)
	require.Equal(t, response.Messages[1].Timestamp, msg2.Timestamp)

	response, err = s2.Next(ctx, response)
	require.NoError(t, err)
	require.Len(t, response.Messages, 2)
	require.Equal(t, response.Messages[0].Timestamp, msg3.Timestamp)
	require.Equal(t, response.Messages[1].Timestamp, msg4.Timestamp)

	response, err = s2.Next(ctx, response)
	require.NoError(t, err)
	require.Len(t, response.Messages, 1)
	require.Equal(t, response.Messages[0].Timestamp, msg5.Timestamp)

	// No more records available
	response, err = s2.Next(ctx, response)
	require.NoError(t, err)
	require.Len(t, response.Messages, 0)
}
