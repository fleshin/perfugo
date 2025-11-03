package handlers

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"gorm.io/gorm"

	applog "perfugo/internal/log"
	"perfugo/internal/views/pages"
)

// Signup displays the account creation form and processes new registrations.
func Signup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	applog.Debug(r.Context(), "handling signup request", "method", r.Method, "htmx", isHTMX(r))

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if ActiveSession(r) {
			applog.Debug(r.Context(), "active session detected during signup, redirecting to app")
			redirectToApp(w, r)
			return
		}
		applog.Debug(r.Context(), "rendering signup form")
		renderSignup(w, r, "", "", "")
	case http.MethodPost:
		if sessionManager == nil || database == nil {
			applog.Debug(r.Context(), "registration dependencies unavailable", "hasSession", sessionManager != nil, "hasDatabase", database != nil)
			http.Error(w, "registration not available", http.StatusServiceUnavailable)
			return
		}
		applog.Debug(r.Context(), "parsing signup form submission")
		if err := r.ParseForm(); err != nil {
			applog.Debug(r.Context(), "failed to parse signup form", "error", err)
			http.Error(w, "invalid form submission", http.StatusBadRequest)
			return
		}

		name := strings.TrimSpace(r.PostFormValue("name"))
		email := strings.TrimSpace(r.PostFormValue("email"))
		password := r.PostFormValue("password")
		confirm := r.PostFormValue("confirm_password")

		applog.Debug(r.Context(), "signup form parsed", "email", strings.ToLower(email))

		if email == "" || !strings.Contains(email, "@") {
			applog.Debug(r.Context(), "invalid signup email", "email", email)
			renderSignup(w, r, "Please provide a valid email address.", name, email)
			return
		}
		if len(password) < 8 {
			applog.Debug(r.Context(), "password too short for signup", "length", len(password))
			renderSignup(w, r, "Password must be at least 8 characters long.", name, email)
			return
		}
		if password != confirm {
			applog.Debug(r.Context(), "signup password mismatch")
			renderSignup(w, r, "Passwords do not match.", name, email)
			return
		}

		if _, err := findUserByEmail(r, email); err == nil {
			applog.Debug(r.Context(), "signup attempted with existing email", "email", strings.ToLower(email))
			renderSignup(w, r, "An account with that email already exists.", name, email)
			return
		} else if err != nil && err != gorm.ErrRecordNotFound {
			applog.Error(r.Context(), "failed to check existing user", "error", err)
			renderSignup(w, r, "We couldn't create your account right now. Please try again.", name, email)
			return
		}

		user, err := createUser(r, email, name, password)
		if err != nil {
			applog.Error(r.Context(), "failed to create user", "error", err)
			renderSignup(w, r, "We couldn't create your account right now. Please try again.", name, email)
			return
		}

		applog.Debug(r.Context(), "user created via signup", "userID", user.ID, "email", user.Email)

		if err := establishSession(r, user); err != nil {
			applog.Error(r.Context(), "failed to establish session after signup", "error", err)
			renderSignup(w, r, "We couldn't sign you in after creating your account. Please try again.", name, email)
			return
		}

		applog.Debug(r.Context(), "signup completed successfully", "userID", user.ID)
		redirectToApp(w, r)
	default:
		applog.Debug(r.Context(), "method not allowed for signup", "method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func renderSignup(w http.ResponseWriter, r *http.Request, message, name, email string) {
	var component templ.Component
	if isHTMX(r) {
		applog.Debug(r.Context(), "rendering HTMX signup partial", "messagePresent", message != "")
		component = pages.SignupPartial(message, name, email)
	} else {
		applog.Debug(r.Context(), "rendering full signup page", "messagePresent", message != "")
		component = pages.Signup(message, name, email)
	}

	if err := component.Render(r.Context(), w); err != nil {
		applog.Error(r.Context(), "failed to render signup component", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
