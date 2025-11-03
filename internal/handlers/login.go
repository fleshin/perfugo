package handlers

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"

	applog "perfugo/internal/log"
	"perfugo/internal/views/pages"
)

// Login renders the authentication view and processes sign-in submissions.
func Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	applog.Debug(r.Context(), "handling login request", "method", r.Method, "htmx", isHTMX(r))

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if ActiveSession(r) {
			applog.Debug(r.Context(), "active session detected, redirecting to app")
			redirectToApp(w, r)
			return
		}
		message := ""
		if sessionManager != nil {
			message = sessionManager.PopString(r.Context(), sessionLoginMessageKey)
		}
		applog.Debug(r.Context(), "rendering login form", "messagePresent", message != "")
		renderLogin(w, r, message, "")
	case http.MethodPost:
		if sessionManager == nil || database == nil {
			applog.Debug(r.Context(), "authentication dependencies unavailable", "hasSession", sessionManager != nil, "hasDatabase", database != nil)
			http.Error(w, "authentication not available", http.StatusServiceUnavailable)
			return
		}
		applog.Debug(r.Context(), "parsing login form submission")
		if err := r.ParseForm(); err != nil {
			applog.Debug(r.Context(), "failed to parse login form", "error", err)
			http.Error(w, "invalid form submission", http.StatusBadRequest)
			return
		}
		email := strings.TrimSpace(r.PostFormValue("email"))
		password := r.PostFormValue("password")

		applog.Debug(r.Context(), "login form parsed", "email", strings.ToLower(email))

		if email == "" || password == "" {
			applog.Debug(r.Context(), "login form missing credentials", "emailPresent", email != "", "passwordPresent", password != "")
			renderLogin(w, r, "Email and password are required.", email)
			return
		}

		if !authenticate(w, r, email, password) {
			applog.Debug(r.Context(), "authentication failed", "email", strings.ToLower(email))
			message := ""
			if sessionManager != nil {
				message = sessionManager.PopString(r.Context(), sessionLoginMessageKey)
			}
			if message == "" {
				message = "We were unable to sign you in. Please try again."
			}
			renderLogin(w, r, message, email)
			return
		}

		applog.Debug(r.Context(), "authentication succeeded", "email", strings.ToLower(email))
		redirectToApp(w, r)
	default:
		applog.Debug(r.Context(), "method not allowed for login", "method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func renderLogin(w http.ResponseWriter, r *http.Request, message, email string) {
	var component templ.Component
	if isHTMX(r) {
		applog.Debug(r.Context(), "rendering HTMX login partial", "messagePresent", message != "")
		component = pages.LoginPartial(message, email)
	} else {
		applog.Debug(r.Context(), "rendering full login page", "messagePresent", message != "")
		component = pages.Login(message, email)
	}

	if err := component.Render(r.Context(), w); err != nil {
		applog.Error(r.Context(), "failed to render login component", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
