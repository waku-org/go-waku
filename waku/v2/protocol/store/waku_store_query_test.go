package store

import (
	"testing"

	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/require"
)

func TestStoreQuery(t *testing.T) {
	defaultPubSubTopic := "test"
	defaultContentTopic := "1"

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: defaultContentTopic,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	msg2 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: "2",
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	s := NewWakuStore(true, nil)
	s.storeMessage(defaultPubSubTopic, msg)
	s.storeMessage(defaultPubSubTopic, msg2)

	response := s.FindMessages(&pb.HistoryQuery{
		ContentFilters: []*pb.ContentFilter{
			{
				ContentTopic: defaultContentTopic,
			},
		},
	})

	require.Len(t, response.Messages, 1)
	require.Equal(t, msg, response.Messages[0])
}

func TestStoreQueryMultipleContentFilters(t *testing.T) {
	defaultPubSubTopic := "test"
	topic1 := "1"
	topic2 := "2"
	topic3 := "3"

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic1,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	msg2 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic2,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	msg3 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic3,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	s := NewWakuStore(true, nil)
	s.storeMessage(defaultPubSubTopic, msg)
	s.storeMessage(defaultPubSubTopic, msg2)
	s.storeMessage(defaultPubSubTopic, msg3)

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
	require.Contains(t, response.Messages, msg)
	require.Contains(t, response.Messages, msg3)
	require.NotContains(t, response.Messages, msg2)
}

func TestStoreQueryPubsubTopicFilter(t *testing.T) {
	topic1 := "1"
	topic2 := "2"
	topic3 := "3"
	pubsubTopic1 := "topic1"
	pubsubTopic2 := "topic2"

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic1,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	msg2 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic2,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	msg3 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic3,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	s := NewWakuStore(true, nil)
	s.storeMessage(pubsubTopic1, msg)
	s.storeMessage(pubsubTopic2, msg2)
	s.storeMessage(pubsubTopic2, msg3)

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
	require.Equal(t, msg, response.Messages[0])
}

func TestStoreQueryPubsubTopicNoMatch(t *testing.T) {
	topic1 := "1"
	topic2 := "2"
	topic3 := "3"
	pubsubTopic1 := "topic1"
	pubsubTopic2 := "topic2"

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic1,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	msg2 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic2,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	msg3 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic3,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	s := NewWakuStore(true, nil)
	s.storeMessage(pubsubTopic2, msg)
	s.storeMessage(pubsubTopic2, msg2)
	s.storeMessage(pubsubTopic2, msg3)

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

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic1,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	msg2 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic2,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	msg3 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic3,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	s := NewWakuStore(true, nil)
	s.storeMessage(pubsubTopic1, msg)
	s.storeMessage(pubsubTopic1, msg2)
	s.storeMessage(pubsubTopic1, msg3)

	response := s.FindMessages(&pb.HistoryQuery{
		PubsubTopic: pubsubTopic1,
	})

	require.Len(t, response.Messages, 3)
	require.Contains(t, response.Messages, msg)
	require.Contains(t, response.Messages, msg2)
	require.Contains(t, response.Messages, msg3)
}

func TestStoreQueryForwardPagination(t *testing.T) {
	topic1 := "1"
	pubsubTopic1 := "topic1"

	s := NewWakuStore(true, nil)
	for i := 0; i < 10; i++ {
		msg := &pb.WakuMessage{
			Payload:      []byte{byte(i)},
			ContentTopic: topic1,
			Version:      0,
			Timestamp:    utils.GetUnixEpoch(),
		}
		s.storeMessage(pubsubTopic1, msg)

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

	s := NewWakuStore(true, nil)
	for i := 0; i < 10; i++ {
		msg := &pb.WakuMessage{
			Payload:      []byte{byte(i)},
			ContentTopic: topic1,
			Version:      0,
			Timestamp:    utils.GetUnixEpoch(),
		}
		s.storeMessage(pubsubTopic1, msg)

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
