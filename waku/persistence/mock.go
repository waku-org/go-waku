package persistence

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // Blank import to register the postgres driver
)

// var dbUrlTemplate = "postgres://postgres@localhost:%s/%s?sslmode=disable"
var dbUrlTemplate = "postgres://harshjain@localhost:%s/%s?sslmode=disable"

func ResetDefaultTestPostgresDB(dropDBUrl string) error {
	db, err := sql.Open("postgres", dropDBUrl)
	if err != nil {
		return err
	}

	deletePrevConnectionsSql := `
	SELECT pid, pg_terminate_backend(pid)
	FROM pg_stat_activity
	WHERE datname in ('template1', 'postgres') AND pid <> pg_backend_pid();`
	_, err = db.Exec(deletePrevConnectionsSql)
	if err != nil {
		return err
	}

	_, err = db.Exec("DROP DATABASE IF EXISTS postgres;")
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE DATABASE postgres;")
	return err
}

func NewMockPgDB() *sql.DB {
	mockPgDBPort := os.Getenv("TEST_DB_PORT")

	//
	dropDBUrl := fmt.Sprintf(dbUrlTemplate, mockPgDBPort, "template1")
	fmt.Println(dropDBUrl)
	if err := ResetDefaultTestPostgresDB(dropDBUrl); err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	mockDBUrl := fmt.Sprintf(dbUrlTemplate, mockPgDBPort, "postgres")
	db, err := sql.Open("pgx", mockDBUrl)
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	return db
}
