package store

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"google.golang.org/protobuf/proto"

	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestStoreQuery(t *testing.T) {
	defaultPubSubTopic := "test"
	defaultContentTopic := "1"

	msg1 := tests.CreateWakuMessage(defaultContentTopic, utils.GetUnixEpoch())
	msg2 := tests.CreateWakuMessage("2", utils.GetUnixEpoch())

	s := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	_ = s.storeMessage(protocol.NewEnvelope(msg1, utils.GetUnixEpoch(), defaultPubSubTopic))
	_ = s.storeMessage(protocol.NewEnvelope(msg2, utils.GetUnixEpoch(), defaultPubSubTopic))

	response := s.FindMessages(&pb.HistoryQuery{
		ContentFilters: []*pb.ContentFilter{
			{
				ContentTopic: defaultContentTopic,
			},
		},
	})

	require.Len(t, response.Messages, 1)
	require.True(t, proto.Equal(msg1, response.Messages[0]))
}

func TestStoreQueryMultipleContentFilters(t *testing.T) {
	defaultPubSubTopic := "test"
	topic1 := "1"
	topic2 := "2"
	topic3 := "3"

	msg1 := tests.CreateWakuMessage(topic1, utils.GetUnixEpoch())
	msg2 := tests.CreateWakuMessage(topic2, utils.GetUnixEpoch())
	msg3 := tests.CreateWakuMessage(topic3, utils.GetUnixEpoch())

	s := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())

	_ = s.storeMessage(protocol.NewEnvelope(msg1, utils.GetUnixEpoch(), defaultPubSubTopic))
	_ = s.storeMessage(protocol.NewEnvelope(msg2, utils.GetUnixEpoch(), defaultPubSubTopic))
	_ = s.storeMessage(protocol.NewEnvelope(msg3, utils.GetUnixEpoch(), defaultPubSubTopic))

	response := s.FindMessages(&pb.HistoryQuery{
		ContentFilters: []*pb.ContentFilter{
			{
				ContentTopic: topic1,
			},
			{
				ContentTopic: topic3,
			},
		},
	})

	require.Len(t, response.Messages, 2)
	require.True(t, proto.Equal(response.Messages[0], msg1))
	require.True(t, proto.Equal(response.Messages[1], msg3))
}

func TestStoreQueryPubsubTopicFilter(t *testing.T) {
	topic1 := "1"
	topic2 := "2"
	topic3 := "3"
	pubsubTopic1 := "topic1"
	pubsubTopic2 := "topic2"

	msg1 := tests.CreateWakuMessage(topic1, utils.GetUnixEpoch())
	msg2 := tests.CreateWakuMessage(topic2, utils.GetUnixEpoch())
	msg3 := tests.CreateWakuMessage(topic3, utils.GetUnixEpoch())

	s := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	_ = s.storeMessage(protocol.NewEnvelope(msg1, utils.GetUnixEpoch(), pubsubTopic1))
	_ = s.storeMessage(protocol.NewEnvelope(msg2, utils.GetUnixEpoch(), pubsubTopic2))
	_ = s.storeMessage(protocol.NewEnvelope(msg3, utils.GetUnixEpoch(), pubsubTopic2))

	response := s.FindMessages(&pb.HistoryQuery{
		PubsubTopic: pubsubTopic1,
		ContentFilters: []*pb.ContentFilter{
			{
				ContentTopic: topic1,
			},
			{
				ContentTopic: topic3,
			},
		},
	})

	require.Len(t, response.Messages, 1)
	require.True(t, proto.Equal(msg1, response.Messages[0]))
}

func TestStoreQueryPubsubTopicNoMatch(t *testing.T) {
	topic1 := "1"
	topic2 := "2"
	topic3 := "3"
	pubsubTopic1 := "topic1"
	pubsubTopic2 := "topic2"

	msg1 := tests.CreateWakuMessage(topic1, utils.GetUnixEpoch())
	msg2 := tests.CreateWakuMessage(topic2, utils.GetUnixEpoch())
	msg3 := tests.CreateWakuMessage(topic3, utils.GetUnixEpoch())

	s := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	_ = s.storeMessage(protocol.NewEnvelope(msg1, utils.GetUnixEpoch(), pubsubTopic2))
	_ = s.storeMessage(protocol.NewEnvelope(msg2, utils.GetUnixEpoch(), pubsubTopic2))
	_ = s.storeMessage(protocol.NewEnvelope(msg3, utils.GetUnixEpoch(), pubsubTopic2))

	response := s.FindMessages(&pb.HistoryQuery{
		PubsubTopic: pubsubTopic1,
	})

	require.Len(t, response.Messages, 0)
}

func TestStoreQueryPubsubTopicAllMessages(t *testing.T) {
	topic1 := "1"
	topic2 := "2"
	topic3 := "3"
	pubsubTopic1 := "topic1"

	msg1 := tests.CreateWakuMessage(topic1, utils.GetUnixEpoch())
	msg2 := tests.CreateWakuMessage(topic2, utils.GetUnixEpoch())
	msg3 := tests.CreateWakuMessage(topic3, utils.GetUnixEpoch())

	s := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	_ = s.storeMessage(protocol.NewEnvelope(msg1, utils.GetUnixEpoch(), pubsubTopic1))
	_ = s.storeMessage(protocol.NewEnvelope(msg2, utils.GetUnixEpoch(), pubsubTopic1))
	_ = s.storeMessage(protocol.NewEnvelope(msg3, utils.GetUnixEpoch(), pubsubTopic1))

	response := s.FindMessages(&pb.HistoryQuery{
		PubsubTopic: pubsubTopic1,
	})

	require.Len(t, response.Messages, 3)
	var msgRcvd [3]bool
	for i := 0; i < 3; i++ {
		if proto.Equal(response.Messages[i], msg1) {
			msgRcvd[0] = true
		} else if proto.Equal(response.Messages[i], msg2) {
			msgRcvd[1] = true
		} else if proto.Equal(response.Messages[i], msg3) {
			msgRcvd[2] = true
		} else {
			t.FailNow()
		}
	}
	for i := 0; i < 3; i++ {
		require.True(t, true, msgRcvd[i])

	}
}

func TestStoreQueryForwardPagination(t *testing.T) {
	topic1 := "1"
	pubsubTopic1 := "topic1"

	s := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	for i := 0; i < 10; i++ {
		msg := tests.CreateWakuMessage(topic1, utils.GetUnixEpoch())
		msg.Payload = []byte{byte(i)}
		_ = s.storeMessage(protocol.NewEnvelope(msg, utils.GetUnixEpoch(), pubsubTopic1))
	}

	response := s.FindMessages(&pb.HistoryQuery{
		PubsubTopic: pubsubTopic1,
		PagingInfo: &pb.PagingInfo{
			Direction: pb.PagingInfo_FORWARD,
		},
	})

	require.Len(t, response.Messages, 10)
	for i := 0; i < 10; i++ {
		require.Equal(t, byte(i), response.Messages[i].Payload[0])
	}
}

func TestStoreQueryBackwardPagination(t *testing.T) {
	topic1 := "1"
	pubsubTopic1 := "topic1"

	s := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	for i := 0; i < 10; i++ {
		msg := &wpb.WakuMessage{
			Payload:      []byte{byte(i)},
			ContentTopic: topic1,
			Version:      0,
			Timestamp:    utils.GetUnixEpoch(),
		}
		_ = s.storeMessage(protocol.NewEnvelope(msg, utils.GetUnixEpoch(), pubsubTopic1))

	}

	response := s.FindMessages(&pb.HistoryQuery{
		PubsubTopic: pubsubTopic1,
		PagingInfo: &pb.PagingInfo{
			Direction: pb.PagingInfo_FORWARD,
		},
	})

	require.Len(t, response.Messages, 10)
	for i := 9; i >= 0; i-- {
		require.Equal(t, byte(i), response.Messages[i].Payload[0])
	}
}

func TestTemporalHistoryQueries(t *testing.T) {
	s := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())

	var messages []*wpb.WakuMessage
	now := utils.GetUnixEpoch()
	for i := 0; i < 10; i++ {
		contentTopic := "1"
		if i%2 == 0 {
			contentTopic = "2"
		}
		msg := tests.CreateWakuMessage(contentTopic, now+int64(i))
		_ = s.storeMessage(protocol.NewEnvelope(msg, utils.GetUnixEpoch(), "test"))
		messages = append(messages, msg)
	}

	// handle temporal history query with a valid time window
	response := s.FindMessages(&pb.HistoryQuery{
		ContentFilters: []*pb.ContentFilter{{ContentTopic: "1"}},
		StartTime:      now + 2,
		EndTime:        now + 5,
	})

	require.Len(t, response.Messages, 2)
	require.Equal(t, messages[3].Timestamp, response.Messages[0].Timestamp)
	require.Equal(t, messages[5].Timestamp, response.Messages[1].Timestamp)

	// handle temporal history query with a zero-size time window
	response = s.FindMessages(&pb.HistoryQuery{
		ContentFilters: []*pb.ContentFilter{{ContentTopic: "1"}},
		StartTime:      now + 2,
		EndTime:        now + 2,
	})

	require.Len(t, response.Messages, 0)

	// handle temporal history query with an invalid time window
	response = s.FindMessages(&pb.HistoryQuery{
		ContentFilters: []*pb.ContentFilter{{ContentTopic: "1"}},
		StartTime:      now + 5,
		EndTime:        now + 2,
	})
	// time window is invalid since start time > end time
	// perhaps it should return an error?

	require.Len(t, response.Messages, 0)
}
