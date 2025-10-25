package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	applog "perfugo/internal/log"
)

const (
	sessionAuthenticatedKey = "auth:authenticated"
	sessionLoginMessageKey  = "auth:message"
	sessionStateKeyPrefix   = "auth:oidc:state:"
	sessionNonceKeyPrefix   = "auth:oidc:nonce:"
)

var (
	sessionManager   *scs.SessionManager
	providerRegistry = map[string]OIDCProvider{}
	providerOrder    []string
)

// OIDCProvider stores the runtime configuration for an OpenID Connect provider.
type OIDCProvider struct {
	ID           string
	DisplayName  string
	OAuth2Config *oauth2.Config
	Verifier     *oidc.IDTokenVerifier
}

// Configure installs the shared dependencies used by the HTTP handlers.
func Configure(sm *scs.SessionManager, providers []OIDCProvider) {
	sessionManager = sm

	providerRegistry = make(map[string]OIDCProvider, len(providers))
	providerOrder = make([]string, 0, len(providers))
	for _, provider := range providers {
		providerRegistry[provider.ID] = provider
		providerOrder = append(providerOrder, provider.ID)
	}
}

// OIDCStartHandler begins the OAuth2 authorization code flow for the configured provider.
func OIDCStartHandler(providerID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		provider, ok := providerRegistry[providerID]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if sessionManager == nil {
			http.Error(w, "authentication not available", http.StatusServiceUnavailable)
			return
		}

		state, err := randomToken()
		if err != nil {
			applog.Error(r.Context(), "failed to generate oidc state", "error", err)
			http.Error(w, "failed to initiate login", http.StatusInternalServerError)
			return
		}
		nonce, err := randomToken()
		if err != nil {
			applog.Error(r.Context(), "failed to generate oidc nonce", "error", err)
			http.Error(w, "failed to initiate login", http.StatusInternalServerError)
			return
		}

		sessionManager.Put(r.Context(), stateSessionKey(providerID), state)
		sessionManager.Put(r.Context(), nonceSessionKey(providerID), nonce)

		authURL := provider.OAuth2Config.AuthCodeURL(state, oauth2.SetAuthURLParam("nonce", nonce))
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

// OIDCCallbackHandler completes the OAuth2 authorization code flow and creates the user session.
func OIDCCallbackHandler(providerID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		provider, ok := providerRegistry[providerID]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if sessionManager == nil {
			http.Error(w, "authentication not available", http.StatusServiceUnavailable)
			return
		}

		if !validateState(r, providerID) {
			sessionManager.Put(r.Context(), sessionLoginMessageKey, "We couldn't verify that login attempt. Please try again.")
			redirectToLogin(w, r)
			return
		}

		code := strings.TrimSpace(r.URL.Query().Get("code"))
		if code == "" {
			sessionManager.Put(r.Context(), sessionLoginMessageKey, "The login response was missing required information. Please try again.")
			redirectToLogin(w, r)
			return
		}

		token, err := provider.OAuth2Config.Exchange(r.Context(), code)
		if err != nil {
			applog.Error(r.Context(), "oidc token exchange failed", "error", err)
			sessionManager.Put(r.Context(), sessionLoginMessageKey, "We couldn't complete the sign in process. Please try again.")
			redirectToLogin(w, r)
			return
		}

		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok || rawIDToken == "" {
			applog.Error(r.Context(), "oidc response missing id_token")
			sessionManager.Put(r.Context(), sessionLoginMessageKey, "We couldn't complete the sign in process. Please try again.")
			redirectToLogin(w, r)
			return
		}

		idToken, err := provider.Verifier.Verify(r.Context(), rawIDToken)
		if err != nil {
			applog.Error(r.Context(), "failed to verify id_token", "error", err)
			sessionManager.Put(r.Context(), sessionLoginMessageKey, "We couldn't verify your sign in. Please try again.")
			redirectToLogin(w, r)
			return
		}

		expectedNonce := sessionManager.PopString(r.Context(), nonceSessionKey(providerID))
		if expectedNonce != "" && idToken.Nonce != expectedNonce {
			applog.Error(r.Context(), "oidc nonce mismatch", "expected", expectedNonce, "actual", idToken.Nonce)
			sessionManager.Put(r.Context(), sessionLoginMessageKey, "We couldn't verify your sign in. Please try again.")
			redirectToLogin(w, r)
			return
		}

		var claims customClaims
		if err := idToken.Claims(&claims); err != nil {
			applog.Error(r.Context(), "failed to parse id_token claims", "error", err)
			sessionManager.Put(r.Context(), sessionLoginMessageKey, "We couldn't verify your sign in. Please try again.")
			redirectToLogin(w, r)
			return
		}

		if err := sessionManager.RenewToken(r.Context()); err != nil {
			applog.Error(r.Context(), "failed to renew session token", "error", err)
			sessionManager.Put(r.Context(), sessionLoginMessageKey, "We couldn't verify your sign in. Please try again.")
			redirectToLogin(w, r)
			return
		}

		sessionManager.Put(r.Context(), sessionAuthenticatedKey, true)
		sessionManager.Put(r.Context(), "auth:user:provider", provider.DisplayName)
		if claims.Email != "" {
			sessionManager.Put(r.Context(), "auth:user:email", claims.Email)
		}
		if claims.Name != "" {
			sessionManager.Put(r.Context(), "auth:user:name", claims.Name)
		}

		redirectToApp(w, r)
	}
}

// RequireAuthentication ensures the user has an active session before accessing the resource.
func RequireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sessionManager == nil || !sessionManager.GetBool(r.Context(), sessionAuthenticatedKey) {
			redirectToLogin(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Logout destroys the current session and redirects the user to the login screen.
func Logout(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if sessionManager != nil {
		if err := sessionManager.Destroy(r.Context()); err != nil {
			applog.Error(r.Context(), "failed to destroy session", "error", err)
		}
	}

	redirectToLogin(w, r)
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func redirectToApp(w http.ResponseWriter, r *http.Request) {
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/app")
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

func validateState(r *http.Request, providerID string) bool {
	expected := sessionManager.PopString(r.Context(), stateSessionKey(providerID))
	if expected == "" {
		return false
	}
	received := strings.TrimSpace(r.URL.Query().Get("state"))
	return received != "" && received == expected
}

func stateSessionKey(providerID string) string {
	return sessionStateKeyPrefix + providerID
}

func nonceSessionKey(providerID string) string {
	return sessionNonceKeyPrefix + providerID
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// AvailableProviders exposes the configured provider identifiers to other handlers.
func AvailableProviders() []OIDCProvider {
	providers := make([]OIDCProvider, 0, len(providerOrder))
	for _, id := range providerOrder {
		if provider, ok := providerRegistry[id]; ok {
			providers = append(providers, provider)
		}
	}
	return providers
}

// ActiveSession returns true when the current request has an authenticated session.
func ActiveSession(r *http.Request) bool {
	return sessionManager != nil && sessionManager.GetBool(r.Context(), sessionAuthenticatedKey)
}

// SessionValue retrieves a value from the session.
func SessionValue[T any](r *http.Request, key string) (T, error) {
	var zero T
	if sessionManager == nil {
		return zero, errors.New("session manager not configured")
	}
	value, ok := sessionManager.Get(r.Context(), key).(T)
	if !ok {
		return zero, fmt.Errorf("session value %q missing or wrong type", key)
	}
	return value, nil
}

type customClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}
