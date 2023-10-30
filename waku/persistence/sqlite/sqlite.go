package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/mattn/go-sqlite3" // Blank import to register the sqlite3 driver
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/persistence/migrate"
	"github.com/waku-org/go-waku/waku/persistence/sqlite/migrations"
	"go.uber.org/zap"
)

func addSqliteURLDefaults(dburl string) string {
	if !strings.Contains(dburl, "?") {
		dburl += "?"
	}

	if !strings.Contains(dburl, "_journal=") {
		dburl += "&_journal=WAL"
	}

	if !strings.Contains(dburl, "_timeout=") {
		dburl += "&_timeout=5000"
	}

	return dburl
}

func executeVacuum(db *sql.DB, logger *zap.Logger) error {
	logger.Info("starting sqlite database vacuuming")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)

	go func() {
		defer cancel()
		_, err := db.Exec("VACUUM")
		if err != nil {
			errCh <- err
		}
	}()

	t := time.NewTicker(2 * time.Minute)
	defer t.Stop()

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case err := <-errCh:
			return err
		case <-t.C:
			logger.Info("still vacuuming...")
		}
	}

	logger.Info("finished sqlite database vacuuming")
	return nil
}

// NewDB creates a sqlite3 DB in the specified path
func NewDB(dburl string, shouldVacuum bool, logger *zap.Logger) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", addSqliteURLDefaults(dburl))
	if err != nil {
		return nil, err
	}

	// Disable concurrent access as not supported by the driver
	db.SetMaxOpenConns(1)

	if shouldVacuum {
		err := executeVacuum(db, logger)
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}

func migrationDriver(db *sql.DB) (database.Driver, error) {
	return sqlite3.WithInstance(db, &sqlite3.Config{
		MigrationsTable: "gowaku_" + sqlite3.DefaultMigrationsTable,
	})
}

// Migrations is the function used for DB migration with sqlite driver
func Migrations(db *sql.DB) error {
	migrationDriver, err := migrationDriver(db)
	if err != nil {
		return err
	}
	return migrate.Migrate(db, migrationDriver, migrations.AssetNames(), migrations.Asset)
}

// CreateTable creates the table that will persist the peers
func CreateTable(db *sql.DB, tableName string) error {
	sqlStmt := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (key TEXT NOT NULL UNIQUE, data BLOB);", tableName)
	_, err := db.Exec(sqlStmt)
	if err != nil {
		return err
	}
	return nil
}

// NewQueries creates a table if it doesn't exist and a new SQL set of queries for the passed table
func NewQueries(tbl string, db *sql.DB) (*persistence.Queries, error) {
	err := CreateTable(db, tbl)
	if err != nil {
		return nil, err
	}
	return persistence.CreateQueries(tbl), nil
}
