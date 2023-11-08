package protocol

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestEnvelope(t *testing.T) {
	e := NewEnvelope(
		&pb.WakuMessage{ContentTopic: "ContentTopic"},
		*utils.GetUnixEpoch(),
		"test",
	)

	msg := e.Message()
	require.Equal(t, "ContentTopic", msg.ContentTopic)

	topic := e.PubsubTopic()
	require.Equal(t, "test", topic)
	hash := e.Hash()
	fmt.Println(hash)

	require.Equal(
		t,
		[]uint8{70, 218, 246, 174, 188, 127, 199, 220, 111, 30, 61, 218, 238, 60, 83, 3, 179, 98, 85, 35, 7, 107, 188, 138, 32, 70, 170, 126, 55, 21, 71, 70},
		hash,
	)
}
