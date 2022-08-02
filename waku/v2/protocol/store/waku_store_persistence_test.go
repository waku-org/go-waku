package store

import (
	"testing"

	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/require"
)

func TestStorePersistence(t *testing.T) {
	db := MemoryDB(t)

	s1 := NewWakuStore(nil, nil, db, utils.Logger())

	defaultPubSubTopic := "test"
	defaultContentTopic := "1"
	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: defaultContentTopic,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	_ = s1.storeMessage(protocol.NewEnvelope(msg, utils.GetUnixEpoch(), defaultPubSubTopic))

	allMsgs, err := db.GetAll()
	require.NoError(t, err)
	require.Len(t, allMsgs, 1)
	require.Equal(t, msg, allMsgs[0].Message)

	// Storing a duplicated message should not crash. It's okay to generate an error log in this case
	err = s1.storeMessage(protocol.NewEnvelope(msg, utils.GetUnixEpoch(), defaultPubSubTopic))
	require.Error(t, err)
}
