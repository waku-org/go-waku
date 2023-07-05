package sqlite

import (
	"database/sql"
	"strings"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/mattn/go-sqlite3" // Blank import to register the sqlite3 driver
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/persistence/migrate"
	"github.com/waku-org/go-waku/waku/persistence/sqlite/migrations"
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

// WithDB is a DBOption that lets you use a sqlite3 DBStore and run migrations
func WithDB(dburl string, migrate bool) persistence.DBOption {
	return func(d *persistence.DBStore) error {
		driverOption := persistence.WithDriver("sqlite3", addSqliteURLDefaults(dburl), persistence.ConnectionPoolOptions{
			// Disable concurrent access as not supported by the driver
			MaxOpenConnections: 1,
		})
		err := driverOption(d)
		if err != nil {
			return err
		}

		if !migrate {
			return nil
		}

		migrationOpt := persistence.WithMigrations(Migrate)
		err = migrationOpt(d)
		if err != nil {
			return err
		}

		return nil
	}
}

// NewDB creates a sqlite3 DB in the specified path
func NewDB(dburl string) (*sql.DB, func(*sql.DB) error, error) {
	db, err := sql.Open("sqlite3", addSqliteURLDefaults(dburl))
	if err != nil {
		return nil, nil, err
	}

	// Disable concurrent access as not supported by the driver
	db.SetMaxOpenConns(1)

	return db, Migrate, nil
}

func migrationDriver(db *sql.DB) (database.Driver, error) {
	return sqlite3.WithInstance(db, &sqlite3.Config{
		MigrationsTable: "gowaku_" + sqlite3.DefaultMigrationsTable,
	})
}

// Migrate is the function used for DB migration with sqlite driver
func Migrate(db *sql.DB) error {
	migrationDriver, err := migrationDriver(db)
	if err != nil {
		return err
	}
	return migrate.Migrate(db, migrationDriver, migrations.AssetNames(), migrations.Asset)
}
