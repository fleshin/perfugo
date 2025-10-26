package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"perfugo/internal/handlers"
	"perfugo/models"
)

func TestNewAppliesSessionDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	if err := db.Create(&models.User{Email: "user@example.com", PasswordHash: string(hash), Theme: models.DefaultTheme}).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	cfg := Config{Addr: ":8080", Session: SessionConfig{CookieSecure: true}, Database: db}
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	t.Cleanup(func() {
		handlers.Configure(nil, nil)
	})

	if srv.httpServer.Addr != ":8080" {
		t.Fatalf("expected server addr :8080, got %q", srv.httpServer.Addr)
	}
	if srv.httpServer.Handler == nil {
		t.Fatal("expected handler to be configured")
	}

	data := url.Values{}
	data.Set("email", "user@example.com")
	data.Set("password", "password123")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after login, got %d", rr.Code)
	}
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie to be set")
	}
	if cookies[0].Name != "perfugo_session" {
		t.Fatalf("expected default session cookie name, got %q", cookies[0].Name)
	}
	if !cookies[0].Secure {
		t.Fatal("expected cookie secure flag to be true")
	}
}

func TestServerHandler(t *testing.T) {
	cfg := Config{Addr: ":9090"}
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		handlers.Configure(nil, nil)
	})

	handler := srv.Handler()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected /healthz to return 200, got %d", rr.Code)
	}
}
