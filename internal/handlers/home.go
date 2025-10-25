package handlers

import (
	"net/http"

	"perfugo/internal/views/pages"
)

// Home renders the primary dashboard experience using templ components.
func Home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.Dashboard().Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
