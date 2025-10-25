package handlers

import (
	"net/http"

	templpkg "github.com/a-h/templ"
	applog "perfugo/internal/log"
	"perfugo/internal/views/pages"
)

// Dashboard renders the main application workspace once a user is authenticated.
func Dashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		applog.Debug(r.Context(), "dashboard access with unsupported method", "method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	applog.Debug(r.Context(), "rendering dashboard", "htmx", isHTMX(r))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var component templpkg.Component
	if isHTMX(r) {
		applog.Debug(r.Context(), "rendering HTMX dashboard partial")
		component = pages.DashboardPartial()
	} else {
		applog.Debug(r.Context(), "rendering full dashboard page")
		component = pages.Dashboard()
	}

	if err := component.Render(r.Context(), w); err != nil {
		applog.Error(r.Context(), "failed to render dashboard", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
