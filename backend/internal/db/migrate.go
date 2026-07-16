package db

import (
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"log"
)

// RunMigrations applies any pending migrations.
func (db *DB) RunMigrations() error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://./migrations",
		"postgres", driver)
	if err != nil {
		log.Printf("migrate.NewWithDatabaseInstance error: %v", err)
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Printf("migrate.Up error: %v", err)
		return err
	}
	return nil
}