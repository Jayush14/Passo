package main

import (
	"log"
	"net/http"
	"time"

	"passo/backend/internal/apikey"
	"passo/backend/internal/auth"
	"passo/backend/internal/chat"
	"passo/backend/internal/config"
	"passo/backend/internal/db"
	"passo/backend/internal/processor"
	"passo/backend/internal/roadmap"
	"passo/backend/internal/server"
)

// gemini package no longer needs DB access - using direct API key in GenerateText function

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database connection
	dbConn, err := db.New(cfg.DBURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// Note: dbConn embeds *sql.DB, so we can call Close() directly.
	defer dbConn.Close()

	// Run migrations
	if err := dbConn.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Set the shared DB instance for auth handlers
	auth.SetDB(dbConn)
	// Set the shared DB instance for apikey handlers
	apikey.SetDB(dbConn)
	// Set the shared DB instance for chat handlers
	chat.SetDB(dbConn)
	// Set the shared DB instance for roadmap handlers
	roadmap.SetDB(dbConn)
	// Set the shared DB instance for processor handlers
	processor.SetDB(dbConn)
	// Gemini no longer requires DB access - using direct API key in GenerateText function

	// Create server
	srv := server.NewServer(cfg)

	// Determine address
	addr := ":" + cfg.Port
	if cfg.Env == "development" {
		log.Printf("Starting server in %s mode on %s", cfg.Env, addr)
	} else {
		log.Printf("Starting server in %s mode on %s", cfg.Env, addr)
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      srv.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}