package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"perfugo/models"
)

func withTestSessionManager(t *testing.T) (*scs.SessionManager, func()) {
	t.Helper()
	original := sessionManager
	sm := scs.New()
	sessionManager = sm
	return sm, func() {
		sessionManager = original
	}
}

func withTestDatabase(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	original := database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}
	database = db
	return db, func() {
		database = original
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	}
}

func TestIsHTMX(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if isHTMX(req) {
		t.Fatal("expected false when no HTMX headers present")
	}
	req.Header.Set("HX-Request", "true")
	if !isHTMX(req) {
		t.Fatal("expected true when HX-Request header present")
	}
}

func TestWorkspaceSectionFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
	}{
		{"/app", ""},
		{"/app/", ""},
		{"/app/formulas", "formulas"},
		{"/app/formulas/123", "formulas/123"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			if got := workspaceSectionFromPath(tt.path); got != tt.want {
				t.Fatalf("workspaceSectionFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestActiveSession(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if ActiveSession(req) {
		t.Fatal("expected inactive session when manager is nil")
	}

	sm, cleanup := withTestSessionManager(t)
	t.Cleanup(cleanup)

	ctx, err := sm.Load(req.Context(), "")
	if err != nil {
		t.Fatalf("failed to load session context: %v", err)
	}
	req = req.WithContext(ctx)
	sm.Put(req.Context(), sessionAuthenticatedKey, true)
	sm.Put(req.Context(), sessionUserIDKey, 42)

	if !ActiveSession(req) {
		t.Fatal("expected active session when flags are set")
	}
}

func TestCurrentUserID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, ok := currentUserID(req); ok {
		t.Fatal("expected currentUserID to fail without session manager")
	}

	sm, cleanup := withTestSessionManager(t)
	t.Cleanup(cleanup)

	ctx, err := sm.Load(req.Context(), "")
	if err != nil {
		t.Fatalf("failed to load session context: %v", err)
	}
	req = req.WithContext(ctx)

	if _, ok := currentUserID(req); ok {
		t.Fatal("expected false when user id not set")
	}

	sm.Put(req.Context(), sessionUserIDKey, 7)
	id, ok := currentUserID(req)
	if !ok || id != 7 {
		t.Fatalf("expected user id 7, got %d (ok=%t)", id, ok)
	}
}

func TestEstablishSession(t *testing.T) {
	sm, cleanup := withTestSessionManager(t)
	t.Cleanup(cleanup)

	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	ctx, err := sm.Load(req.Context(), "")
	if err != nil {
		t.Fatalf("failed to load session context: %v", err)
	}
	req = req.WithContext(ctx)

	user := &models.User{Model: gorm.Model{ID: 3}, Email: "user@example.com", Name: "User"}
	if err := establishSession(req, user); err != nil {
		t.Fatalf("establishSession returned error: %v", err)
	}

	if !sm.GetBool(req.Context(), sessionAuthenticatedKey) {
		t.Fatal("expected session authenticated flag to be true")
	}
	if got := sm.GetInt(req.Context(), sessionUserIDKey); got != 3 {
		t.Fatalf("expected session user id 3, got %d", got)
	}
	if got := sm.GetString(req.Context(), sessionUserEmailKey); got != "user@example.com" {
		t.Fatalf("unexpected email %q", got)
	}
	if got := sm.GetString(req.Context(), sessionUserNameKey); got != "User" {
		t.Fatalf("unexpected name %q", got)
	}
}

func TestEstablishSessionWithoutManager(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	if err := establishSession(req, &models.User{}); err == nil {
		t.Fatal("expected error when session manager is nil")
	}
}

func TestCreateUser(t *testing.T) {
	db, dbCleanup := withTestDatabase(t)
	t.Cleanup(dbCleanup)

	req := httptest.NewRequest(http.MethodPost, "/signup", nil)
	user, err := createUser(req, "Example@Email.com", "  Test User  ", "password123")
	if err != nil {
		t.Fatalf("createUser returned error: %v", err)
	}
	if user.Email != "example@email.com" {
		t.Fatalf("expected email to be lowercased, got %q", user.Email)
	}
	if user.Name != "Test User" {
		t.Fatalf("expected trimmed name, got %q", user.Name)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("password123")); err != nil {
		t.Fatalf("password hash does not match original: %v", err)
	}

	var count int64
	if err := db.Model(&models.User{}).Where("email = ?", "example@email.com").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("expected user persisted, count=%d err=%v", count, err)
	}
}

func TestCreateUserWithoutDatabase(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/signup", nil)
	if _, err := createUser(req, "test@example.com", "User", "password"); !errors.Is(err, gorm.ErrInvalidDB) {
		t.Fatalf("expected ErrInvalidDB, got %v", err)
	}
}

func TestFindUserByEmail(t *testing.T) {
	_, cleanup := withTestSessionManager(t)
	t.Cleanup(cleanup)

	_, dbCleanup := withTestDatabase(t)
	t.Cleanup(dbCleanup)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, err := findUserByEmail(req, "missing@example.com"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound for missing user, got %v", err)
	}

	if _, err := createUser(req, "user@example.com", "User", "password123"); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	user, err := findUserByEmail(req, "USER@example.com")
	if err != nil {
		t.Fatalf("findUserByEmail returned error: %v", err)
	}
	if user.Email != "user@example.com" {
		t.Fatalf("expected lowercase email, got %q", user.Email)
	}
}

func TestAuthenticate(t *testing.T) {
	sm, smCleanup := withTestSessionManager(t)
	t.Cleanup(smCleanup)
	_, dbCleanup := withTestDatabase(t)
	t.Cleanup(dbCleanup)

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	ctx, err := sm.Load(req.Context(), "")
	if err != nil {
		t.Fatalf("failed to load session context: %v", err)
	}
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	if _, err := createUser(req, "user@example.com", "User", "password123"); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if ok := authenticate(w, req, "user@example.com", "password123"); !ok {
		t.Fatal("expected authentication to succeed")
	}
	if !sm.GetBool(req.Context(), sessionAuthenticatedKey) {
		t.Fatal("expected session authenticated flag to be true")
	}

	w = httptest.NewRecorder()
	if ok := authenticate(w, req, "user@example.com", "wrong"); ok {
		t.Fatal("expected authentication failure with bad password")
	}
	if message := sm.PopString(req.Context(), sessionLoginMessageKey); message == "" {
		t.Fatal("expected login failure message to be set")
	}
}

func TestRedirectToLogin(t *testing.T) {
	_, cleanup := withTestSessionManager(t)
	t.Cleanup(cleanup)

	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	redirectToLogin(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 for HTMX redirect, got %d", w.Code)
	}
	if w.Header().Get("HX-Redirect") != "/login" {
		t.Fatalf("expected HX-Redirect header to be set")
	}

	req = httptest.NewRequest(http.MethodGet, "/app", nil)
	w = httptest.NewRecorder()
	redirectToLogin(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/login" {
		t.Fatalf("expected redirect to /login, got %q", loc)
	}
}

func TestRedirectToApp(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.Header.Set("HX-Boosted", "true")
	w := httptest.NewRecorder()
	redirectToApp(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 status, got %d", w.Code)
	}
	if w.Header().Get("HX-Redirect") != "/app" {
		t.Fatalf("expected HX-Redirect header to be set")
	}

	req = httptest.NewRequest(http.MethodGet, "/login", nil)
	w = httptest.NewRecorder()
	redirectToApp(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 status, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/app" {
		t.Fatalf("expected redirect to /app, got %q", loc)
	}
}

func TestLoadCurrentUserTheme(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	if theme := loadCurrentUserTheme(req); theme != models.DefaultTheme {
		t.Fatalf("expected default theme when no dependencies, got %q", theme)
	}

	sm, smCleanup := withTestSessionManager(t)
	t.Cleanup(smCleanup)
	ctx, err := sm.Load(req.Context(), "")
	if err != nil {
		t.Fatalf("failed to load session context: %v", err)
	}
	req = req.WithContext(ctx)
	sm.Put(req.Context(), sessionUserThemeKey, "midnight_draft")
	if theme := loadCurrentUserTheme(req); theme != models.ThemeMidnightDraft {
		t.Fatalf("expected normalized theme from session, got %q", theme)
	}

	db, dbCleanup := withTestDatabase(t)
	t.Cleanup(dbCleanup)
	sm.Put(req.Context(), sessionUserThemeKey, "")

	user := &models.User{Email: "user@example.com", Theme: "atelier_ivory"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	sm.Put(req.Context(), sessionUserIDKey, int(user.ID))
	if theme := loadCurrentUserTheme(req); theme != models.ThemeAtelierIvory {
		t.Fatalf("expected theme from database, got %q", theme)
	}
	if cached := sm.GetString(req.Context(), sessionUserThemeKey); cached != models.ThemeAtelierIvory {
		t.Fatalf("expected theme to be cached in session, got %q", cached)
	}
}
