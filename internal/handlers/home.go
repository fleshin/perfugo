package handlers

import (
	"net/http"
)

// Home renders the landing experience for the perfumery platform.
func Home(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/assets/index.html", 302)
}
