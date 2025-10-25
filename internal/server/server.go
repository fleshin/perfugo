package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"gorm.io/gorm"

	"perfugo/internal/handlers"
	applog "perfugo/internal/log"
)

// Config captures the runtime configuration for the HTTP server.
type Config struct {
	Addr     string
	Session  SessionConfig
	Database *gorm.DB
}

// SessionConfig controls session behavior for the HTTP server.
type SessionConfig struct {
	Lifetime     time.Duration
	CookieName   string
	CookieDomain string
	CookieSecure bool
}

// Server wraps an http.Server and exposes helpers for bootstrapping a
// production-ready web service.
type Server struct {
	config     Config
	httpServer *http.Server
}

// New builds a new Server using the provided configuration.
func New(cfg Config) (*Server, error) {
	applog.Debug(context.Background(), "initializing server",
		"addr", cfg.Addr,
		"sessionLifetime", cfg.Session.Lifetime.String(),
		"sessionCookie", cfg.Session.CookieName,
	)

	sessionCfg := cfg.Session
	if sessionCfg.Lifetime <= 0 {
		applog.Debug(context.Background(), "session lifetime not provided, using default")
		sessionCfg.Lifetime = 12 * time.Hour
	}
	if strings.TrimSpace(sessionCfg.CookieName) == "" {
		applog.Debug(context.Background(), "session cookie name not provided, using default")
		sessionCfg.CookieName = "perfugo_session"
	}

	sessionManager := scs.New()
	sessionManager.Lifetime = sessionCfg.Lifetime
	sessionManager.Cookie.Name = sessionCfg.CookieName
	sessionManager.Cookie.Domain = sessionCfg.CookieDomain
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Secure = sessionCfg.CookieSecure

	applog.Debug(context.Background(), "session manager configured",
		"cookieName", sessionCfg.CookieName,
		"cookieDomain", sessionCfg.CookieDomain,
		"cookieSecure", sessionCfg.CookieSecure,
	)

	handlers.Configure(sessionManager, cfg.Database)

	applog.Debug(context.Background(), "handler dependencies configured")

	handler := sessionManager.LoadAndSave(newRouter())

	applog.Debug(context.Background(), "http handler chain prepared")

	return &Server{
		config: cfg,
		httpServer: &http.Server{
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}, nil
}

// Start begins serving HTTP traffic using the underlying http.Server.
func (s *Server) Start() error {
	applog.Debug(context.Background(), "server starting listener", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the HTTP server with a timeout.
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	applog.Debug(ctx, "server initiating graceful shutdown")
	return s.httpServer.Shutdown(ctx)
}

// Handler exposes the configured HTTP handler, enabling integration tests.
func (s *Server) Handler() http.Handler {
	applog.Debug(context.Background(), "server handler requested")
	return s.httpServer.Handler
}
