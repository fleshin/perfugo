package handlers

import (
	"net/http"
	"strings"

	templpkg "github.com/a-h/templ"
	"gorm.io/gorm"

	applog "perfugo/internal/log"
	"perfugo/internal/views/pages"
)

// Signup displays the account creation form and processes new registrations.
func Signup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if ActiveSession(r) {
			redirectToApp(w, r)
			return
		}
		renderSignup(w, r, "", "", "")
	case http.MethodPost:
		if sessionManager == nil || database == nil {
			http.Error(w, "registration not available", http.StatusServiceUnavailable)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form submission", http.StatusBadRequest)
			return
		}

		name := strings.TrimSpace(r.PostFormValue("name"))
		email := strings.TrimSpace(r.PostFormValue("email"))
		password := r.PostFormValue("password")
		confirm := r.PostFormValue("confirm_password")

		if email == "" || !strings.Contains(email, "@") {
			renderSignup(w, r, "Please provide a valid email address.", name, email)
			return
		}
		if len(password) < 8 {
			renderSignup(w, r, "Password must be at least 8 characters long.", name, email)
			return
		}
		if password != confirm {
			renderSignup(w, r, "Passwords do not match.", name, email)
			return
		}

		if _, err := findUserByEmail(r, email); err == nil {
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

		if err := establishSession(r, user); err != nil {
			applog.Error(r.Context(), "failed to establish session after signup", "error", err)
			renderSignup(w, r, "We couldn't sign you in after creating your account. Please try again.", name, email)
			return
		}

		redirectToApp(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func renderSignup(w http.ResponseWriter, r *http.Request, message, name, email string) {
	var component templpkg.Component
	if isHTMX(r) {
		component = pages.SignupPartial(message, name, email)
	} else {
		component = pages.Signup(message, name, email)
	}

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
