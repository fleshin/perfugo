package handlers

import (
	"net/http"

	templpkg "github.com/a-h/templ"

	"perfugo/internal/views/pages"
)

// Home renders the landing experience for the perfumery platform.
func Home(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var component templpkg.Component
	if isHTMX(r) {
		component = pages.LandingPartial()
	} else {
		component = pages.Landing()
	}

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
