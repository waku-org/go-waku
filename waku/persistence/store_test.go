package persistence

import (
	"database/sql"
	"testing"
)

func SetStoreDB(t *testing.T, db *sql.DB, d *DBStore) {
	d.db = db
}

func SetMaxMessages(t *testing.T, m int, d *DBStore) {
	d.maxMessages = m
}
