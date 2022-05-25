package persistence

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3" // Blank import to register the sqlite3 driver
	"github.com/status-im/go-waku/tests"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func NewMock() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		utils.Logger().Fatal("opening a stub database connection", zap.Error(err))
	}

	return db
}

func createIndex(digest []byte, receiverTime int64) *pb.Index {
	return &pb.Index{
		Digest:       digest,
		ReceiverTime: receiverTime,
		SenderTime:   1.0,
	}
}

func TestDbStore(t *testing.T) {
	db := NewMock()
	option := WithDB(db)
	store, err := NewDBStore(utils.Logger().Sugar(), option)
	require.NoError(t, err)

	res, err := store.GetAll()
	require.NoError(t, err)
	require.Empty(t, res)

	err = store.Put(
		createIndex([]byte("digest"), 1),
		"test",
		tests.CreateWakuMessage("test", 1),
	)
	require.NoError(t, err)

	res, err = store.GetAll()
	require.NoError(t, err)
	require.NotEmpty(t, res)
}

func TestStoreRetention(t *testing.T) {
	db := NewMock()
	store, err := NewDBStore(utils.Logger().Sugar(), WithDB(db), WithRetentionPolicy(5, 20*time.Second))
	require.NoError(t, err)

	insertTime := time.Now()

	_ = store.Put(createIndex([]byte{1}, insertTime.Add(-70*time.Second).UnixNano()), "test", tests.CreateWakuMessage("test", 1))
	_ = store.Put(createIndex([]byte{2}, insertTime.Add(-60*time.Second).UnixNano()), "test", tests.CreateWakuMessage("test", 2))
	_ = store.Put(createIndex([]byte{3}, insertTime.Add(-50*time.Second).UnixNano()), "test", tests.CreateWakuMessage("test", 3))
	_ = store.Put(createIndex([]byte{4}, insertTime.Add(-40*time.Second).UnixNano()), "test", tests.CreateWakuMessage("test", 4))
	_ = store.Put(createIndex([]byte{5}, insertTime.Add(-30*time.Second).UnixNano()), "test", tests.CreateWakuMessage("test", 5))

	dbResults, err := store.GetAll()
	require.NoError(t, err)
	require.Len(t, dbResults, 5)

	_ = store.Put(createIndex([]byte{6}, insertTime.Add(-20*time.Second).UnixNano()), "test", tests.CreateWakuMessage("test", 6))
	_ = store.Put(createIndex([]byte{7}, insertTime.Add(-10*time.Second).UnixNano()), "test", tests.CreateWakuMessage("test", 7))

	// This step simulates starting go-waku again from scratch

	store, err = NewDBStore(utils.Logger().Sugar(), WithDB(db), WithRetentionPolicy(5, 40*time.Second))
	require.NoError(t, err)

	dbResults, err = store.GetAll()
	require.NoError(t, err)
	require.Len(t, dbResults, 3)
	require.Equal(t, []byte{5}, dbResults[0].ID)
	require.Equal(t, []byte{6}, dbResults[1].ID)
	require.Equal(t, []byte{7}, dbResults[2].ID)
}
