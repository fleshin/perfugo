package handlers

import (
	"net/http"
	"strings"

	templpkg "github.com/a-h/templ"

	"perfugo/internal/views/pages"
)

// Login renders the authentication view and processes sign-in submissions.
func Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if ActiveSession(r) {
			redirectToApp(w, r)
			return
		}
		message := ""
		if sessionManager != nil {
			message = sessionManager.PopString(r.Context(), sessionLoginMessageKey)
		}
		renderLogin(w, r, message, "")
	case http.MethodPost:
		if sessionManager == nil || database == nil {
			http.Error(w, "authentication not available", http.StatusServiceUnavailable)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form submission", http.StatusBadRequest)
			return
		}
		email := strings.TrimSpace(r.PostFormValue("email"))
		password := r.PostFormValue("password")

		if email == "" || password == "" {
			renderLogin(w, r, "Email and password are required.", email)
			return
		}

		if !authenticate(w, r, email, password) {
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

		redirectToApp(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func renderLogin(w http.ResponseWriter, r *http.Request, message, email string) {
	var component templpkg.Component
	if isHTMX(r) {
		component = pages.LoginPartial(message, email)
	} else {
		component = pages.Login(message, email)
	}

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
