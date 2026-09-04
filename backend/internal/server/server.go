package server

import (
	"net/http"

	"passo/backend/internal/config"
	"passo/backend/internal/db"
	"passo/backend/internal/plugins/market-data-engine/api/handlers"
	"passo/backend/internal/plugins/market-data-engine/processing"
)

// Server holds server configuration.
type Server struct {
	Cfg *config.Config
	DB  *db.DB
}

// NewServer creates a new Server instance.
func NewServer(cfg *config.Config, dbInst *db.DB) *Server {
	return &Server{
		Cfg: cfg,
		DB:  dbInst,
	}
}

// Router returns the HTTP request multiplexer with all routes registered.
func (s *Server) Router() *http.ServeMux {
	// Initialize MDE worker pool
	workerConfig := processing.WorkerConfig{}
	workerPool := processing.NewWorkerPool(s.DB.Pool, workerConfig)

	// Initialize MDE processing handler with the worker pool
	mdeHandler := handlers.NewProcessingHandler(workerPool)

	// Update the router to use the properly initialized handler
	return RouterWithMDEHandler(mdeHandler)
}
