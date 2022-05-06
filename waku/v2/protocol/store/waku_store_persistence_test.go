package store

import (
	"context"
	"database/sql"
	"testing"

	"github.com/status-im/go-waku/waku/persistence"
	"github.com/status-im/go-waku/waku/persistence/sqlite"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/require"
)

func TestStorePersistence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var db *sql.DB
	db, err := sqlite.NewDB(":memory:")
	require.NoError(t, err)

	dbStore, err := persistence.NewDBStore(utils.Logger(), persistence.WithDB(db))
	require.NoError(t, err)

	s1 := NewWakuStore(nil, nil, dbStore, 0, 0, utils.Logger())
	s1.fetchDBRecords(ctx)
	require.Len(t, s1.messageQueue.messages, 0)

	defaultPubSubTopic := "test"
	defaultContentTopic := "1"
	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: defaultContentTopic,
		Version:      0,
		Timestamp:    utils.GetUnixEpoch(),
	}

	_ = s1.storeMessage(protocol.NewEnvelope(msg, defaultPubSubTopic))

	s2 := NewWakuStore(nil, nil, dbStore, 0, 0, utils.Logger())
	s2.fetchDBRecords(ctx)
	require.Len(t, s2.messageQueue.messages, 1)
	require.Equal(t, msg, s2.messageQueue.messages[0].msg)

	// Storing a duplicated message should not crash. It's okay to generate an error log in this case
	err = s1.storeMessage(protocol.NewEnvelope(msg, defaultPubSubTopic))
	require.ErrorIs(t, err, ErrDuplicatedMessage)
}
