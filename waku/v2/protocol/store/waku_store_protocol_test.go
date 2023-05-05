package store

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestWakuStoreProtocolQuery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(MemoryDB(t), timesource.NewDefaultClock(), utils.Logger())
	s1.SetHost(host1)

	topic1 := "1"
	pubsubTopic1 := "topic1"

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic1,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	// Simulate a message has been received via relay protocol
	sub := relay.ArraySubscription([]*protocol.Envelope{protocol.NewEnvelope(msg, utils.GetUnixEpoch(), pubsubTopic1)})
	err = s1.Start(ctx, sub)
	require.NoError(t, err)
	defer s1.Stop()

	s2 := NewWakuStore(MemoryDB(t), timesource.NewDefaultClock(), utils.Logger())
	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)
	s2.SetHost(host2)
	err = s2.Start(ctx, relay.NoopSubscription())
	require.NoError(t, err)
	defer s2.Stop()

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), StoreID_v20beta4)
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

func TestWakuStoreProtocolLocalQuery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(MemoryDB(t), timesource.NewDefaultClock(), utils.Logger())
	s1.SetHost(host1)

	topic1 := "1"
	pubsubTopic1 := "topic1"

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic1,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	// Simulate a message has been received via relay protocol
	sub := relay.ArraySubscription([]*protocol.Envelope{protocol.NewEnvelope(msg, utils.GetUnixEpoch(), pubsubTopic1)})
	err = s1.Start(ctx, sub)
	require.NoError(t, err)
	defer s1.Stop()

	time.Sleep(100 * time.Millisecond)

	q := Query{
		Topic:         pubsubTopic1,
		ContentTopics: []string{topic1},
	}
	response, err := s1.Query(ctx, q, WithLocalQuery())

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
	s1 := NewWakuStore(db, timesource.NewDefaultClock(), utils.Logger())
	s1.SetHost(host1)

	topic1 := "1"
	pubsubTopic1 := "topic1"

	now := utils.GetUnixEpoch()
	msg1 := tests.CreateWakuMessage(topic1, now+1)
	msg2 := tests.CreateWakuMessage(topic1, now+2)
	msg3 := tests.CreateWakuMessage(topic1, now+3)
	msg4 := tests.CreateWakuMessage(topic1, now+4)
	msg5 := tests.CreateWakuMessage(topic1, now+5)

	sub := relay.ArraySubscription([]*protocol.Envelope{
		protocol.NewEnvelope(msg1, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg2, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg3, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg4, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg5, utils.GetUnixEpoch(), pubsubTopic1),
	})
	err = s1.Start(ctx, sub)
	require.NoError(t, err)

	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), StoreID_v20beta4)
	require.NoError(t, err)

	s2 := NewWakuStore(MemoryDB(t), timesource.NewDefaultClock(), utils.Logger())
	s2.SetHost(host2)
	err = s2.Start(ctx, relay.NoopSubscription())
	require.NoError(t, err)
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
	require.True(t, response.started)
	require.Equal(t, response.Messages[0].Timestamp, msg3.Timestamp)
	require.Equal(t, response.Messages[1].Timestamp, msg4.Timestamp)

	response, err = s2.Next(ctx, response)
	require.NoError(t, err)
	require.True(t, response.started)
	require.Len(t, response.Messages, 1)
	require.Equal(t, response.Messages[0].Timestamp, msg5.Timestamp)

	// No more records available
	response, err = s2.Next(ctx, response)
	require.NoError(t, err)
	require.True(t, response.started)
	require.Len(t, response.Messages, 0)
}

func TestWakuStoreResult(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	db := MemoryDB(t)
	s1 := NewWakuStore(db, timesource.NewDefaultClock(), utils.Logger())
	s1.SetHost(host1)

	topic1 := "1"
	pubsubTopic1 := "topic1"

	now := utils.GetUnixEpoch()
	msg1 := tests.CreateWakuMessage(topic1, now+1)
	msg2 := tests.CreateWakuMessage(topic1, now+2)
	msg3 := tests.CreateWakuMessage(topic1, now+3)
	msg4 := tests.CreateWakuMessage(topic1, now+4)
	msg5 := tests.CreateWakuMessage(topic1, now+5)

	sub := relay.ArraySubscription([]*protocol.Envelope{
		protocol.NewEnvelope(msg1, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg2, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg3, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg4, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg5, utils.GetUnixEpoch(), pubsubTopic1),
	})
	err = s1.Start(ctx, sub)
	require.NoError(t, err)

	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), StoreID_v20beta4)
	require.NoError(t, err)

	s2 := NewWakuStore(MemoryDB(t), timesource.NewDefaultClock(), utils.Logger())
	s2.SetHost(host2)
	err = s2.Start(ctx, relay.NoopSubscription())
	require.NoError(t, err)
	defer s2.Stop()

	q := Query{
		Topic:         pubsubTopic1,
		ContentTopics: []string{topic1},
	}

	result, err := s2.Query(ctx, q, WithAutomaticPeerSelection(), WithAutomaticRequestId(), WithPaging(true, 2))
	require.NoError(t, err)
	require.False(t, result.started)
	require.Len(t, result.GetMessages(), 0)
	require.False(t, result.IsComplete())

	// First iteration
	hasNext, err := result.Next(ctx)
	require.NoError(t, err)
	require.True(t, result.started)
	require.True(t, hasNext)
	require.Len(t, result.GetMessages(), 2)
	require.Equal(t, result.GetMessages()[0].Timestamp, msg1.Timestamp)
	require.Equal(t, result.GetMessages()[1].Timestamp, msg2.Timestamp)
	require.False(t, result.IsComplete())

	// Second iteration
	hasNext, err = result.Next(ctx)
	require.NoError(t, err)
	require.True(t, result.started)
	require.True(t, hasNext)
	require.Len(t, result.GetMessages(), 2)
	require.Equal(t, result.GetMessages()[0].Timestamp, msg3.Timestamp)
	require.Equal(t, result.GetMessages()[1].Timestamp, msg4.Timestamp)
	require.False(t, result.IsComplete())

	// Third iteration
	hasNext, err = result.Next(ctx)
	require.NoError(t, err)
	require.True(t, result.started)
	require.True(t, hasNext)
	require.Len(t, result.GetMessages(), 1)
	require.Equal(t, result.GetMessages()[0].Timestamp, msg5.Timestamp)
	require.True(t, result.IsComplete())

	// No more records available
	hasNext, err = result.Next(ctx)
	require.NoError(t, err)
	require.True(t, result.started)
	require.False(t, hasNext)
}

func TestWakuStoreProtocolFind(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(MemoryDB(t), timesource.NewDefaultClock(), utils.Logger())
	s1.SetHost(host1)

	topic1 := "1"
	pubsubTopic1 := "topic1"

	now := utils.GetUnixEpoch()
	msg1 := tests.CreateWakuMessage(topic1, now+1)
	msg2 := tests.CreateWakuMessage(topic1, now+2)
	msg3 := tests.CreateWakuMessage(topic1, now+3)
	msg4 := tests.CreateWakuMessage(topic1, now+4)
	msg5 := tests.CreateWakuMessage(topic1, now+5)
	msg6 := tests.CreateWakuMessage(topic1, now+6)
	msg7 := tests.CreateWakuMessage("hello", now+7)
	msg8 := tests.CreateWakuMessage(topic1, now+8)
	msg9 := tests.CreateWakuMessage(topic1, now+9)

	sub := relay.ArraySubscription([]*protocol.Envelope{
		protocol.NewEnvelope(msg1, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg2, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg3, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg4, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg5, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg6, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg7, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg8, utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg9, utils.GetUnixEpoch(), pubsubTopic1),
	})
	err = s1.Start(ctx, sub)
	require.NoError(t, err)
	defer s1.Stop()

	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), StoreID_v20beta4)
	require.NoError(t, err)

	s2 := NewWakuStore(MemoryDB(t), timesource.NewDefaultClock(), utils.Logger())
	s2.SetHost(host2)
	err = s2.Start(ctx, relay.NoopSubscription())
	require.NoError(t, err)
	defer s2.Stop()

	q := Query{
		Topic: pubsubTopic1,
	}

	fn := func(msg *pb.WakuMessage) (bool, error) {
		return msg.ContentTopic == "hello", nil
	}

	foundMsg, err := s2.Find(ctx, q, fn, WithAutomaticPeerSelection(), WithAutomaticRequestId(), WithPaging(true, 2))
	require.NoError(t, err)
	require.NotNil(t, foundMsg)
	require.Equal(t, "hello", foundMsg.ContentTopic)

	fn2 := func(msg *pb.WakuMessage) (bool, error) {
		return msg.ContentTopic == "bye", nil
	}

	foundMsg, err = s2.Find(ctx, q, fn2, WithAutomaticPeerSelection(), WithAutomaticRequestId(), WithPaging(true, 2))
	require.NoError(t, err)
	require.Nil(t, foundMsg)
}
