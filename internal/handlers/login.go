package handlers

import (
	"net/http"

	templpkg "github.com/a-h/templ"
	"perfugo/internal/views/pages"
)

// Login renders the authentication view and processes basic HTMX submissions.
func Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	switch r.Method {
	case http.MethodGet, http.MethodPost:
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	message := ""
	if r.Method == http.MethodPost {
		message = "Authentication will be available soon. Request access from your studio administrator."
	}

	var component templpkg.Component
	if isHTMX(r) {
		component = pages.LoginPartial(message)
	} else {
		component = pages.Login(message)
	}

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
