package handlers

import (
	"net/http"

	templpkg "github.com/a-h/templ"
	"perfugo/internal/views/pages"
)

// Dashboard renders the main application workspace once a user is authenticated.
func Dashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var component templpkg.Component
	if isHTMX(r) {
		component = pages.DashboardPartial()
	} else {
		component = pages.Dashboard()
	}

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
