package server

import (
	"context"
	"net/http"
	"time"
)

// Config captures the runtime configuration for the HTTP server.
type Config struct {
	Addr string
}

// Server wraps an http.Server and exposes helpers for bootstrapping a
// production-ready web service.
type Server struct {
	config     Config
	httpServer *http.Server
}

// New builds a new Server using the provided configuration.
func New(cfg Config) *Server {
	handler := newRouter()
	return &Server{
		config: cfg,
		httpServer: &http.Server{
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Start begins serving HTTP traffic using the underlying http.Server.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the HTTP server with a timeout.
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// Handler exposes the configured HTTP handler, enabling integration tests.
func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}
