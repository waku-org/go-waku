package utils

import (
	"database/sql"
	"errors"
	"regexp"
	"strings"

	"github.com/waku-org/go-waku/waku/persistence/postgres"
	"github.com/waku-org/go-waku/waku/persistence/sqlite"
)

func validateDBUrl(val string) error {
	matched, err := regexp.Match(`^[\w\+]+:\/\/[\w\/\\\.\:\@]+\?{0,1}.*$`, []byte(val))
	if !matched || err != nil {
		return errors.New("invalid db url option format")
	}
	return nil
}

// ExtractDBAndMigration will return a database connection, and migration function that should be used depending on a database connection string
func ExtractDBAndMigration(databaseURL string) (*sql.DB, func(*sql.DB) error, error) {
	var db *sql.DB
	var migrationFn func(*sql.DB) error
	var err error

	dbURL := ""
	if databaseURL != "" {
		err := validateDBUrl(databaseURL)
		if err != nil {
			return nil, nil, err
		}
		dbURL = databaseURL
	} else {
		// In memoryDB
		dbURL = "sqlite3://:memory:"
	}

	dbURLParts := strings.Split(dbURL, "://")
	dbEngine := dbURLParts[0]
	dbParams := dbURLParts[1]
	switch dbEngine {
	case "sqlite3":
		db, migrationFn, err = sqlite.NewDB(dbParams)
	case "postgresql":
		db, migrationFn, err = postgres.NewDB(dbURL)
	default:
		err = errors.New("unsupported database engine")
	}

	if err != nil {
		return nil, nil, err
	}

	return db, migrationFn, nil

}
