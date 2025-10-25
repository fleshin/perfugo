package handlers

import (
	"net/http"

	applog "perfugo/internal/log"
)

// Home renders the landing experience for the perfumery platform.
func Home(w http.ResponseWriter, r *http.Request) {
	applog.Debug(r.Context(), "redirecting home request to landing page", "method", r.Method)
	http.Redirect(w, r, "/assets/index.html", 302)
}
