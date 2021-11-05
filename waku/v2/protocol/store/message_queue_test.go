package store

import (
	"testing"

	"github.com/status-im/go-waku/tests"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/stretchr/testify/require"
)

func TestMessageQueue(t *testing.T) {
	msg1 := tests.CreateWakuMessage("1", 1)
	msg2 := tests.CreateWakuMessage("2", 2)
	msg3 := tests.CreateWakuMessage("3", 3)
	msg4 := tests.CreateWakuMessage("3", 3)
	msg5 := tests.CreateWakuMessage("3", 3)

	msgQ := NewMessageQueue(3)
	msgQ.Push(IndexedWakuMessage{msg: msg1, index: &pb.Index{}, pubsubTopic: "test"})
	msgQ.Push(IndexedWakuMessage{msg: msg2, index: &pb.Index{}, pubsubTopic: "test"})
	msgQ.Push(IndexedWakuMessage{msg: msg3, index: &pb.Index{}, pubsubTopic: "test"})

	require.Len(t, msgQ.messages, 3)

	msgQ.Push(IndexedWakuMessage{msg: msg4, index: &pb.Index{}, pubsubTopic: "test"})

	require.Len(t, msgQ.messages, 3)
	require.Equal(t, msg2.Payload, msgQ.messages[0].msg.Payload)
	require.Equal(t, msg4.Payload, msgQ.messages[2].msg.Payload)

	msgQ.Push(IndexedWakuMessage{msg: msg5, index: &pb.Index{}, pubsubTopic: "test"})

	require.Len(t, msgQ.messages, 3)
	require.Equal(t, msg3.Payload, msgQ.messages[0].msg.Payload)
	require.Equal(t, msg5.Payload, msgQ.messages[2].msg.Payload)
}
