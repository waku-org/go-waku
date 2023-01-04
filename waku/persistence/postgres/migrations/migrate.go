//go:build !gowaku_skip_migrations
// +build !gowaku_skip_migrations

package migrations

import (
	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"

	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
)

// Migrate applies migrations.
func Migrate(db *sql.DB, driver database.Driver) error {
	return migrateDB(db, bindata.Resource(
		AssetNames(),
		Asset,
	), driver)
}

// Migrate database using provided resources.
func migrateDB(db *sql.DB, resources *bindata.AssetSource, driver database.Driver) error {
	source, err := bindata.WithInstance(resources)
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"go-bindata",
		source,
		"gowakudb",
		driver)
	if err != nil {
		return err
	}

	if err = m.Up(); err != migrate.ErrNoChange {
		return err
	}
	return nil
}
