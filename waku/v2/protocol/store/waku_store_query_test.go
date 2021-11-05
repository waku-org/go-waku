package store

import (
	"testing"

	"github.com/status-im/go-waku/tests"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/require"
)

func TestStoreQuery(t *testing.T) {
	defaultPubSubTopic := "test"
	defaultContentTopic := "1"

	msg1 := tests.CreateWakuMessage(defaultContentTopic, utils.GetUnixEpoch())
	msg2 := tests.CreateWakuMessage("2", utils.GetUnixEpoch())

	s := NewWakuStore(nil, 0, 0)
	s.storeMessage(protocol.NewEnvelope(msg1, defaultPubSubTopic))
	s.storeMessage(protocol.NewEnvelope(msg2, defaultPubSubTopic))

	response := s.FindMessages(&pb.HistoryQuery{
		ContentFilters: []*pb.ContentFilter{
			{
				ContentTopic: defaultContentTopic,
			},
		},
	})

	require.Len(t, response.Messages, 1)
	require.Equal(t, msg1, response.Messages[0])
}

func TestStoreQueryMultipleContentFilters(t *testing.T) {
	defaultPubSubTopic := "test"
	topic1 := "1"
	topic2 := "2"
	topic3 := "3"

	msg1 := tests.CreateWakuMessage(topic1, utils.GetUnixEpoch())
	msg2 := tests.CreateWakuMessage(topic2, utils.GetUnixEpoch())
	msg3 := tests.CreateWakuMessage(topic3, utils.GetUnixEpoch())

	s := NewWakuStore(nil, 0, 0)

	s.storeMessage(protocol.NewEnvelope(msg1, defaultPubSubTopic))
	s.storeMessage(protocol.NewEnvelope(msg2, defaultPubSubTopic))
	s.storeMessage(protocol.NewEnvelope(msg3, defaultPubSubTopic))

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
	require.Contains(t, response.Messages, msg1)
	require.Contains(t, response.Messages, msg3)
	require.NotContains(t, response.Messages, msg2)
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

	s := NewWakuStore(nil, 0, 0)
	s.storeMessage(protocol.NewEnvelope(msg1, pubsubTopic1))
	s.storeMessage(protocol.NewEnvelope(msg2, pubsubTopic2))
	s.storeMessage(protocol.NewEnvelope(msg3, pubsubTopic2))

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
	require.Equal(t, msg1, response.Messages[0])
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

	s := NewWakuStore(nil, 0, 0)
	s.storeMessage(protocol.NewEnvelope(msg1, pubsubTopic2))
	s.storeMessage(protocol.NewEnvelope(msg2, pubsubTopic2))
	s.storeMessage(protocol.NewEnvelope(msg3, pubsubTopic2))

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

	s := NewWakuStore(nil, 0, 0)
	s.storeMessage(protocol.NewEnvelope(msg1, pubsubTopic1))
	s.storeMessage(protocol.NewEnvelope(msg2, pubsubTopic1))
	s.storeMessage(protocol.NewEnvelope(msg3, pubsubTopic1))

	response := s.FindMessages(&pb.HistoryQuery{
		PubsubTopic: pubsubTopic1,
	})

	require.Len(t, response.Messages, 3)
	require.Contains(t, response.Messages, msg1)
	require.Contains(t, response.Messages, msg2)
	require.Contains(t, response.Messages, msg3)
}

func TestStoreQueryForwardPagination(t *testing.T) {
	topic1 := "1"
	pubsubTopic1 := "topic1"

	s := NewWakuStore(nil, 0, 0)
	for i := 0; i < 10; i++ {
		msg := tests.CreateWakuMessage(topic1, utils.GetUnixEpoch())
		msg.Payload = []byte{byte(i)}
		s.storeMessage(protocol.NewEnvelope(msg, pubsubTopic1))
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

	s := NewWakuStore(nil, 0, 0)
	for i := 0; i < 10; i++ {
		msg := &pb.WakuMessage{
			Payload:      []byte{byte(i)},
			ContentTopic: topic1,
			Version:      0,
			Timestamp:    utils.GetUnixEpoch(),
		}
		s.storeMessage(protocol.NewEnvelope(msg, pubsubTopic1))

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
	s := NewWakuStore(nil, 0, 0)

	var messages []*pb.WakuMessage
	for i := 0; i < 10; i++ {
		contentTopic := "1"
		if i%2 == 0 {
			contentTopic = "2"
		}
		msg := tests.CreateWakuMessage(contentTopic, float64(i))
		s.storeMessage(protocol.NewEnvelope(msg, "test"))
		messages = append(messages, msg)
	}

	// handle temporal history query with a valid time window
	response := s.FindMessages(&pb.HistoryQuery{
		ContentFilters: []*pb.ContentFilter{{ContentTopic: "1"}},
		StartTime:      float64(2),
		EndTime:        float64(5),
	})

	require.Len(t, response.Messages, 2)
	require.Equal(t, messages[3].Timestamp, response.Messages[0].Timestamp)
	require.Equal(t, messages[5].Timestamp, response.Messages[1].Timestamp)

	// handle temporal history query with a zero-size time window
	response = s.FindMessages(&pb.HistoryQuery{
		ContentFilters: []*pb.ContentFilter{{ContentTopic: "1"}},
		StartTime:      float64(2),
		EndTime:        float64(2),
	})

	require.Len(t, response.Messages, 0)

	// handle temporal history query with an invalid time window
	response = s.FindMessages(&pb.HistoryQuery{
		ContentFilters: []*pb.ContentFilter{{ContentTopic: "1"}},
		StartTime:      float64(5),
		EndTime:        float64(2),
	})
	// time window is invalid since start time > end time
	// perhaps it should return an error?

	require.Len(t, response.Messages, 0)
}
