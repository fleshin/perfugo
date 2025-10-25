package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	applog "perfugo/internal/log"
	"perfugo/internal/views/theme"
)

type preferencesResponse struct {
	Theme string `json:"theme"`
}

// UpdatePreferences persists workspace preferences for the authenticated user.
func UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		applog.Debug(r.Context(), "preferences update with unsupported method", "method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user, err := loadCurrentUser(r)
	if err != nil {
		applog.Error(r.Context(), "unable to load current user for preferences", "error", err)
		http.Error(w, "unable to load account", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		applog.Error(r.Context(), "failed to parse preferences form", "error", err)
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}

	themeValue := strings.TrimSpace(r.FormValue("theme"))
	themeConfig := theme.Resolve(themeValue)
	if themeConfig.Key == "" {
		applog.Debug(r.Context(), "received invalid theme selection", "value", themeValue)
		http.Error(w, "invalid theme selection", http.StatusBadRequest)
		return
	}

	if database == nil {
		applog.Debug(r.Context(), "database not configured; skipping preference persistence")
	} else {
		applog.Debug(r.Context(), "updating user preferences", "userID", user.ID, "theme", themeConfig.Key)
		if err := database.WithContext(r.Context()).Model(user).Update("theme", themeConfig.Key).Error; err != nil {
			applog.Error(r.Context(), "failed to persist user preferences", "error", err)
			http.Error(w, "failed to save preferences", http.StatusInternalServerError)
			return
		}
	}

	setSessionTheme(r, themeConfig.Key)

	response := preferencesResponse{Theme: themeConfig.Key}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		applog.Error(r.Context(), "failed to encode preferences response", "error", err)
	}
}
