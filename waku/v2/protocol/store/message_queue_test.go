package store

import (
	"testing"
	"time"

	"github.com/status-im/go-waku/tests"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/require"
)

func TestMessageQueue(t *testing.T) {
	msg1 := tests.CreateWakuMessage("1", 1)
	msg2 := tests.CreateWakuMessage("2", 2)
	msg3 := tests.CreateWakuMessage("3", 3)
	msg4 := tests.CreateWakuMessage("3", 3)
	msg5 := tests.CreateWakuMessage("3", 3)

	msgQ := NewMessageQueue(3, 1*time.Minute)
	msgQ.Push(IndexedWakuMessage{msg: msg1, index: &pb.Index{Digest: []byte{1}, ReceiverTime: utils.GetUnixEpochFrom(time.Now().Add(-20 * time.Second))}, pubsubTopic: "test"})
	msgQ.Push(IndexedWakuMessage{msg: msg2, index: &pb.Index{Digest: []byte{2}, ReceiverTime: utils.GetUnixEpochFrom(time.Now().Add(-15 * time.Second))}, pubsubTopic: "test"})
	msgQ.Push(IndexedWakuMessage{msg: msg3, index: &pb.Index{Digest: []byte{3}, ReceiverTime: utils.GetUnixEpochFrom(time.Now().Add(-10 * time.Second))}, pubsubTopic: "test"})

	require.Equal(t, msgQ.Length(), 3)

	msgQ.Push(IndexedWakuMessage{msg: msg4, index: &pb.Index{Digest: []byte{4}, ReceiverTime: utils.GetUnixEpochFrom(time.Now().Add(-3 * time.Second))}, pubsubTopic: "test"})

	require.Len(t, msgQ.messages, 3)
	require.Equal(t, msg2.Payload, msgQ.messages[0].msg.Payload)
	require.Equal(t, msg4.Payload, msgQ.messages[2].msg.Payload)

	indexedMsg5 := IndexedWakuMessage{msg: msg5, index: &pb.Index{Digest: []byte{5}, ReceiverTime: utils.GetUnixEpochFrom(time.Now().Add(0 * time.Second))}, pubsubTopic: "test"}

	msgQ.Push(indexedMsg5)

	require.Len(t, msgQ.messages, 3)
	require.Equal(t, msg3.Payload, msgQ.messages[0].msg.Payload)
	require.Equal(t, msg5.Payload, msgQ.messages[2].msg.Payload)

	// Test duplication
	msgQ.Push(indexedMsg5)

	require.Len(t, msgQ.messages, 3)
	require.Equal(t, msg3.Payload, msgQ.messages[0].msg.Payload)
	require.Equal(t, msg4.Payload, msgQ.messages[1].msg.Payload)
	require.Equal(t, msg5.Payload, msgQ.messages[2].msg.Payload)

	// Test retention
	msgQ.maxDuration = 5 * time.Second
	msgQ.cleanOlderRecords()
	require.Len(t, msgQ.messages, 2)
	require.Equal(t, msg4.Payload, msgQ.messages[0].msg.Payload)
	require.Equal(t, msg5.Payload, msgQ.messages[1].msg.Payload)
}
