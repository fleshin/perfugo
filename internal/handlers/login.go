package handlers

import (
	"fmt"
	"net/http"

	templpkg "github.com/a-h/templ"
	"perfugo/internal/views/pages"
)

// Login renders the authentication view and processes basic HTMX submissions.
func Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	switch r.Method {
	case http.MethodGet, http.MethodHead:
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if ActiveSession(r) {
		redirectToApp(w, r)
		return
	}

	message := ""
	if sessionManager != nil {
		message = sessionManager.PopString(r.Context(), sessionLoginMessageKey)
	}

	providers := AvailableProviders()
	options := make([]pages.LoginOption, 0, len(providers))
	for _, provider := range providers {
		options = append(options, pages.LoginOption{
			Title:       fmt.Sprintf("Continue with %s", provider.DisplayName),
			Description: fmt.Sprintf("Sign in using your %s account.", provider.DisplayName),
			URL:         "/auth/oidc/" + provider.ID + "/start",
		})
	}

	if len(options) == 0 && message == "" {
		message = "Authentication is not yet configured. Contact your studio administrator for access."
	}

	var component templpkg.Component
	if isHTMX(r) {
		component = pages.LoginPartial(message, options)
	} else {
		component = pages.Login(message, options)
	}

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
