package main

import (
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"passo/backend/internal/config"
	"passo/backend/internal/db"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database connection
	dbConn, err := db.New(cfg.DBURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	// Force migration version 21 to fix dirty state
	driver, err := postgres.WithInstance(dbConn.DB, &postgres.Config{})
	if err != nil {
		log.Fatalf("Failed to create postgres driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://./migrations",
		"postgres", driver)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	// Force the version to 21 (mark as applied)
	if err := m.Force(21); err != nil {
		log.Fatalf("Failed to force version 21: %v", err)
	}

	log.Println("Successfully forced migration version 21")
}