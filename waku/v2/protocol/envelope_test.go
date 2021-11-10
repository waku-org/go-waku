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
		[]uint8([]byte{0x2d, 0x84, 0x3d, 0x43, 0x77, 0x14, 0x4, 0xad, 0x64, 0x9d, 0x90, 0xd6, 0x5c, 0xc7, 0x3d, 0x8f, 0x21, 0x49, 0xa, 0xf1, 0x9c, 0x83, 0x88, 0x76, 0x51, 0xba, 0x6f, 0x34, 0x14, 0x78, 0x93, 0xd2}),
		hash,
	)

	size := e.Size()
	require.Equal(t, 14, size)
}
