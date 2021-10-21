package persistence

import (
	"database/sql"
	"log"
	"testing"

	_ "github.com/mattn/go-sqlite3" // Blank import to register the sqlite3 driver
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/stretchr/testify/require"
)

func NewMock() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	return db
}

func TestDbStore(t *testing.T) {
	db := NewMock()
	option := WithDB(db)
	store, err := NewDBStore(option)
	require.NoError(t, err)

	res, err := store.GetAll()
	require.NoError(t, err)
	require.Empty(t, res)

	err = store.Put(
		&pb.Index{
			Digest:       []byte("digest"),
			ReceiverTime: 1.0,
			SenderTime:   1.0,
		},
		"test",
		&pb.WakuMessage{
			Payload:      []byte("payload"),
			ContentTopic: "contenttopic",
			Version:      1,
			Timestamp:    1.0,
			Proof:        []byte("proof"),
		},
	)
	require.NoError(t, err)

	res, err = store.GetAll()
	require.NoError(t, err)
	require.NotEmpty(t, res)
}
