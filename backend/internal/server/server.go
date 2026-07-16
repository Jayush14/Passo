package server

import (
	"net/http"

	"passo/backend/internal/config"
)

// Server holds server configuration.
type Server struct {
	Cfg *config.Config
}

// NewServer creates a new Server instance.
func NewServer(cfg *config.Config) *Server {
	return &Server{Cfg: cfg}
}

// Router returns the HTTP request multiplexer with all routes registered.
func (s *Server) Router() *http.ServeMux {
	return Router()
}