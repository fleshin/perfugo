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

	ctx := r.Context()

	rawTheme := strings.TrimSpace(r.PostFormValue("theme"))
	applog.Debug(ctx, "preferences theme submission received", "userID", userID, "rawTheme", rawTheme)

	requestedTheme := models.NormalizeTheme(rawTheme)
	applog.Debug(ctx, "preferences theme normalized", "userID", userID, "rawTheme", rawTheme, "normalizedTheme", requestedTheme)

	if err := database.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("theme", requestedTheme).Error; err != nil {
		applog.Error(ctx, "failed to update theme preference", "error", err, "userID", userID)
		http.Error(w, "unable to save preferences", http.StatusInternalServerError)
		return
	}

	applog.Debug(ctx, "workspace theme preference persisted", "userID", userID, "theme", requestedTheme)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(preferencesResponse{Theme: requestedTheme}); err != nil {
		applog.Error(ctx, "failed to encode preferences response", "error", err)
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
	ctx := r.Context()
	theme := models.DefaultTheme
	applog.Debug(ctx, "begin theme resolution", "defaultTheme", theme)

	if sessionManager == nil || database == nil {
		applog.Debug(ctx, "theme resolution dependencies missing", "hasSession", sessionManager != nil, "hasDatabase", database != nil, "resolvedTheme", theme)
		return theme
	}

	userID, ok := currentUserID(r)
	if !ok {
		applog.Debug(ctx, "theme resolution without authenticated user", "resolvedTheme", theme)
		return theme
	}

	applog.Debug(ctx, "loading theme preference", "userID", userID)

	var user models.User
	if err := database.WithContext(ctx).Select("theme").First(&user, userID).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			applog.Error(ctx, "failed to load user theme", "error", err, "userID", userID)
		}
		applog.Debug(ctx, "no stored theme found; using default", "userID", userID, "resolvedTheme", theme)
		return theme
	}

	if user.Theme != "" {
		normalized := models.NormalizeTheme(user.Theme)
		applog.Debug(ctx, "resolved stored theme", "userID", userID, "storedTheme", user.Theme, "normalizedTheme", normalized)
		return normalized
	}

	applog.Debug(ctx, "stored theme empty; using default", "userID", userID, "resolvedTheme", theme)
	return theme
}
