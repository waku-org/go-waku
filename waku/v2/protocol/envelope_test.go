package protocol

import (
	"testing"

	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/stretchr/testify/require"
)

func TestEnvelope(t *testing.T) {
	e := NewEnvelope(
		&pb.WakuMessage{ContentTopic: "ContentTopic"},
		"test",
	)

	msg := e.Message()
	require.Equal(t, "ContentTopic", msg.ContentTopic)

	topic := e.PubsubTopic()
	require.Equal(t, "test", topic)

	hash := e.Hash()
	require.Equal(
		t,
		[]uint8([]byte{0xc7, 0xaf, 0xc3, 0xe9, 0x9, 0xd1, 0xc6, 0xb4, 0x81, 0xb3, 0xdf, 0x4f, 0x16, 0x1a, 0xe4, 0xc9, 0x9c, 0x8, 0x4e, 0x5, 0xe4, 0xeb, 0x5f, 0x9b, 0x58, 0xb5, 0xf4, 0xde, 0xe9, 0x73, 0x18, 0x7b}),
		hash,
	)

	size := e.Size()
	require.Equal(t, 14, size)
}
