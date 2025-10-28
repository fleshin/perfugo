package handlers

import (
	"net/http"
	"strings"

	"gorm.io/gorm"

	applog "perfugo/internal/log"
	"perfugo/internal/views/layout"
	"perfugo/internal/views/pages"
	"perfugo/models"
)

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

	ctx := r.Context()

	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 10); err != nil {
			applog.Debug(ctx, "failed to parse multipart preferences form", "error", err)
			http.Error(w, "invalid form submission", http.StatusBadRequest)
			return
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
	} else {
		if err := r.ParseForm(); err != nil {
			applog.Debug(ctx, "failed to parse preferences form", "error", err)
			http.Error(w, "invalid form submission", http.StatusBadRequest)
			return
		}
	}

	rawTheme := strings.TrimSpace(r.FormValue("theme"))
	if rawTheme == "" && r.MultipartForm != nil {
		if values := r.MultipartForm.Value["theme"]; len(values) > 0 {
			rawTheme = strings.TrimSpace(values[0])
		}
	}
	applog.Debug(ctx, "preferences theme submission received", "userID", userID, "rawTheme", rawTheme)

	requestedTheme := models.NormalizeTheme(rawTheme)
	applog.Debug(ctx, "preferences theme normalized", "userID", userID, "rawTheme", rawTheme, "normalizedTheme", requestedTheme)

	if err := database.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("theme", requestedTheme).Error; err != nil {
		applog.Error(ctx, "failed to update theme preference", "error", err, "userID", userID)
		http.Error(w, "unable to save preferences", http.StatusInternalServerError)
		return
	}

	applog.Debug(ctx, "workspace theme preference persisted", "userID", userID, "theme", requestedTheme)

	sessionManager.Put(ctx, sessionUserThemeKey, requestedTheme)
	applog.Debug(ctx, "session theme updated", "userID", userID, "theme", requestedTheme)

	if !isHTMX(r) {
		http.Redirect(w, r, "/app/preferences", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	component := pages.PreferencesSaved(requestedTheme, layout.ThemeOptions())
	if err := component.Render(ctx, w); err != nil {
		applog.Error(ctx, "failed to render preferences response", "error", err)
		http.Error(w, "unable to render preferences", http.StatusInternalServerError)
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

	if sessionManager == nil {
		applog.Debug(ctx, "theme resolution dependencies missing", "hasSession", false, "resolvedTheme", theme)
		return theme
	}

	storedTheme := sessionManager.GetString(ctx, sessionUserThemeKey)
	if storedTheme != "" {
		normalized := models.NormalizeTheme(storedTheme)
		applog.Debug(ctx, "resolved theme from session", "storedTheme", storedTheme, "normalizedTheme", normalized)
		return normalized
	}
	applog.Debug(ctx, "no theme found in session")

	if database == nil {
		applog.Debug(ctx, "theme resolution dependencies missing", "hasDatabase", false, "resolvedTheme", theme)
		return theme
	}

	userID, ok := currentUserID(r)
	if !ok {
		applog.Debug(ctx, "theme resolution without authenticated user", "resolvedTheme", theme)
		return theme
	}

	applog.Debug(ctx, "loading theme preference from database", "userID", userID)

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
		applog.Debug(ctx, "resolved stored theme from database", "userID", userID, "storedTheme", user.Theme, "normalizedTheme", normalized)
		sessionManager.Put(ctx, sessionUserThemeKey, normalized)
		applog.Debug(ctx, "session theme updated from database value", "userID", userID, "theme", normalized)
		return normalized
	}

	applog.Debug(ctx, "stored theme empty; using default", "userID", userID, "resolvedTheme", theme)
	return theme
}
