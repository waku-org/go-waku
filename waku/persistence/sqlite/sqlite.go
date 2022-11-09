package sqlite

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3" // Blank import to register the sqlite3 driver
	"github.com/waku-org/go-waku/waku/persistence"
)

// Queries are the sqlite queries for a given table.
type Queries struct {
	deleteQuery  string
	existsQuery  string
	getQuery     string
	putQuery     string
	queryQuery   string
	prefixQuery  string
	limitQuery   string
	offsetQuery  string
	getSizeQuery string
}

// NewQueries creates a new SQLite set of queries for the passed table
func NewQueries(tbl string, db *sql.DB) (*Queries, error) {
	err := CreateTable(db, tbl)
	if err != nil {
		return nil, err
	}
	return &Queries{
		deleteQuery:  fmt.Sprintf("DELETE FROM %s WHERE key = $1", tbl),
		existsQuery:  fmt.Sprintf("SELECT exists(SELECT 1 FROM %s WHERE key=$1)", tbl),
		getQuery:     fmt.Sprintf("SELECT data FROM %s WHERE key = $1", tbl),
		putQuery:     fmt.Sprintf("INSERT INTO %s (key, data) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET data = $2", tbl),
		queryQuery:   fmt.Sprintf("SELECT key, data FROM %s", tbl),
		prefixQuery:  ` WHERE key LIKE '%s%%' ORDER BY key`,
		limitQuery:   ` LIMIT %d`,
		offsetQuery:  ` OFFSET %d`,
		getSizeQuery: fmt.Sprintf("SELECT length(data) FROM %s WHERE key = $1", tbl),
	}, nil
}

// Delete returns the query for deleting a row.
func (q Queries) Delete() string {
	return q.deleteQuery
}

// Exists returns the query for determining if a row exists.
func (q Queries) Exists() string {
	return q.existsQuery
}

// Get returns the query for getting a row.
func (q Queries) Get() string {
	return q.getQuery
}

// Put returns the query for putting a row.
func (q Queries) Put() string {
	return q.putQuery
}

// Query returns the query for getting multiple rows.
func (q Queries) Query() string {
	return q.queryQuery
}

// Prefix returns the query fragment for getting a rows with a key prefix.
func (q Queries) Prefix() string {
	return q.prefixQuery
}

// Limit returns the query fragment for limiting results.
func (q Queries) Limit() string {
	return q.limitQuery
}

// Offset returns the query fragment for returning rows from a given offset.
func (q Queries) Offset() string {
	return q.offsetQuery
}

// GetSize returns the query for determining the size of a value.
func (q Queries) GetSize() string {
	return q.getSizeQuery
}

// WithDB is a DBOption that lets you use a sqlite3 DBStore.
func WithDB(path string) persistence.DBOption {
	return persistence.WithDriver("sqlite3", path)
}

// NewDB creates a sqlite3 DB in the specified path
func NewDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// CreateTable creates the table that will persist the peers
func CreateTable(db *sql.DB, tableName string) error {
	sqlStmt := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (key TEXT NOT NULL UNIQUE, data BYTEA);", tableName)
	_, err := db.Exec(sqlStmt)
	if err != nil {
		return err
	}
	return nil
}
