package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"perfugo/internal/handlers"
)

// Config captures the runtime configuration for the HTTP server.
type Config struct {
	Addr    string
	Session SessionConfig
	OIDC    OIDCConfig
}

// SessionConfig controls session behavior for the HTTP server.
type SessionConfig struct {
	Lifetime     time.Duration
	CookieName   string
	CookieDomain string
	CookieSecure bool
}

// OIDCConfig captures the OpenID Connect provider configuration.
type OIDCConfig struct {
	ProviderName string
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// Server wraps an http.Server and exposes helpers for bootstrapping a
// production-ready web service.
type Server struct {
	config     Config
	httpServer *http.Server
}

// New builds a new Server using the provided configuration.
func New(cfg Config) (*Server, error) {
	sessionCfg := cfg.Session
	if sessionCfg.Lifetime <= 0 {
		sessionCfg.Lifetime = 12 * time.Hour
	}
	if strings.TrimSpace(sessionCfg.CookieName) == "" {
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

	providers, err := buildOIDCProviders(cfg.OIDC)
	if err != nil {
		return nil, err
	}
	handlers.Configure(sessionManager, providers)

	handler := sessionManager.LoadAndSave(newRouter(providers))

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

func buildOIDCProviders(cfg OIDCConfig) ([]handlers.OIDCProvider, error) {
	trimmedIssuer := strings.TrimSpace(cfg.Issuer)
	trimmedClientID := strings.TrimSpace(cfg.ClientID)
	trimmedSecret := strings.TrimSpace(cfg.ClientSecret)
	trimmedRedirect := strings.TrimSpace(cfg.RedirectURL)

	if trimmedIssuer == "" && trimmedClientID == "" && trimmedSecret == "" && trimmedRedirect == "" {
		return nil, nil
	}

	if trimmedIssuer == "" || trimmedClientID == "" || trimmedSecret == "" || trimmedRedirect == "" {
		return nil, fmt.Errorf("incomplete OIDC configuration")
	}

	provider, err := oidc.NewProvider(context.Background(), trimmedIssuer)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	scopes := []string{oidc.ScopeOpenID, "profile", "email"}
	seen := map[string]struct{}{oidc.ScopeOpenID: {}, "profile": {}, "email": {}}
	for _, scope := range cfg.Scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		scopes = append(scopes, trimmed)
	}

	oauthCfg := &oauth2.Config{
		ClientID:     trimmedClientID,
		ClientSecret: trimmedSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  trimmedRedirect,
		Scopes:       scopes,
	}

	displayName := strings.TrimSpace(cfg.ProviderName)
	if displayName == "" {
		displayName = "OIDC"
	}
	providerID := strings.ToLower(strings.ReplaceAll(displayName, " ", "-"))

	return []handlers.OIDCProvider{
		{
			ID:           providerID,
			DisplayName:  displayName,
			OAuth2Config: oauthCfg,
			Verifier:     provider.Verifier(&oidc.Config{ClientID: trimmedClientID}),
		},
	}, nil
}
