package store

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/persistence/sqlite"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func MemoryDB(t *testing.T) *persistence.DBStore {
	var db *sql.DB
	db, err := sqlite.NewDB(":memory:")
	require.NoError(t, err)

	dbStore, err := persistence.NewDBStore(utils.Logger(), persistence.WithDB(db))
	require.NoError(t, err)

	return dbStore
}
