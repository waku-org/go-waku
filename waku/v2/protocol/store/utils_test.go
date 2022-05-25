package store

import (
	"database/sql"
	"testing"

	"github.com/status-im/go-waku/waku/persistence"
	"github.com/status-im/go-waku/waku/persistence/sqlite"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/require"
)

func MemoryDB(t *testing.T) *persistence.DBStore {
	var db *sql.DB
	db, err := sqlite.NewDB(":memory:")
	require.NoError(t, err)

	dbStore, err := persistence.NewDBStore(utils.Logger(), persistence.WithDB(db))
	require.NoError(t, err)

	return dbStore
}
