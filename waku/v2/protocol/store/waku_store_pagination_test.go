package store

import (
	"testing"

	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/require"
)

func TestIndexComputation(t *testing.T) {
	msg := &pb.WakuMessage{
		Payload:   []byte{1, 2, 3},
		Timestamp: utils.GetUnixEpoch(),
	}

	idx, err := computeIndex(msg)
	require.NoError(t, err)
	require.NotZero(t, idx.ReceiverTime)
	require.Equal(t, msg.Timestamp, idx.SenderTime)
	require.NotZero(t, idx.Digest)
	require.Len(t, idx.Digest, 32)

	msg1 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		Timestamp:    123,
		ContentTopic: "/waku/2/default-content/proto",
	}
	idx1, err := computeIndex(msg1)
	require.NoError(t, err)

	msg2 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		Timestamp:    123,
		ContentTopic: "/waku/2/default-content/proto",
	}
	idx2, err := computeIndex(msg2)
	require.NoError(t, err)

	require.Equal(t, idx1.Digest, idx2.Digest)
}

// TODO: "Index comparison, IndexedWakuMessage comparison, and Sorting tests"
// TODO: "Find Index test"
// TODO: "Forward pagination test"
// TODO: "Backward pagination test"
