package sqlite

import (
	"database/sql"
	"log"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/persistence"
)

func NewMock() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	return db
}

func TestQueries(t *testing.T) {
	db := NewMock()
	queries, err := persistence.NewQueries("test_queries", db)
	require.NoError(t, err)

	query := queries.Delete()
	require.NotEmpty(t, query)

	query = queries.Exists()
	require.NotEmpty(t, query)

	query = queries.Get()
	require.NotEmpty(t, query)

	query = queries.Put()
	require.NotEmpty(t, query)

	query = queries.Query()
	require.NotEmpty(t, query)

	query = queries.Prefix()
	require.NotEmpty(t, query)

	query = queries.Limit()
	require.NotEmpty(t, query)

	query = queries.Offset()
	require.NotEmpty(t, query)

	query = queries.GetSize()
	require.NotEmpty(t, query)
}

func TestCreateTable(t *testing.T) {
	db := NewMock()

	err := persistence.CreateTable(db, "test_create_table")
	require.NoError(t, err)
}
