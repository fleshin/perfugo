package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"gorm.io/gorm"

	applog "perfugo/internal/log"
	"perfugo/models"
)

type preferencesResponse struct {
	Theme string `json:"theme"`
}

// Preferences updates the authenticated user's saved workspace preferences.
func Preferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Dashboard(w, r)
		return
	}

	if sessionManager == nil || database == nil {
		applog.Debug(r.Context(), "preferences dependencies unavailable", "hasSession", sessionManager != nil, "hasDatabase", database != nil)
		http.Error(w, "preferences not available", http.StatusServiceUnavailable)
		return
	}

	userID, ok := currentUserID(r)
	if !ok {
		applog.Debug(r.Context(), "preferences request missing authenticated user")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		applog.Debug(r.Context(), "failed to parse preferences form", "error", err)
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}

	requestedTheme := models.NormalizeTheme(strings.TrimSpace(r.PostFormValue("theme")))

	if err := database.WithContext(r.Context()).Model(&models.User{}).Where("id = ?", userID).Update("theme", requestedTheme).Error; err != nil {
		applog.Error(r.Context(), "failed to update theme preference", "error", err, "userID", userID)
		http.Error(w, "unable to save preferences", http.StatusInternalServerError)
		return
	}

	applog.Debug(r.Context(), "workspace preferences saved", "userID", userID, "theme", requestedTheme)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(preferencesResponse{Theme: requestedTheme}); err != nil {
		applog.Error(r.Context(), "failed to encode preferences response", "error", err)
	}
}

func currentUserID(r *http.Request) (uint, bool) {
	if sessionManager == nil {
		return 0, false
	}
	id := sessionManager.GetInt(r.Context(), sessionUserIDKey)
	if id <= 0 {
		return 0, false
	}
	return uint(id), true
}

func loadCurrentUserTheme(r *http.Request) string {
	theme := models.DefaultTheme
	if sessionManager == nil || database == nil {
		return theme
	}
	userID, ok := currentUserID(r)
	if !ok {
		return theme
	}

	var user models.User
	if err := database.WithContext(r.Context()).Select("theme").First(&user, userID).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			applog.Error(r.Context(), "failed to load user theme", "error", err, "userID", userID)
		}
		return theme
	}

	if user.Theme != "" {
		return models.NormalizeTheme(user.Theme)
	}
	return theme
}
