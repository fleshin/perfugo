package config

import (
	"testing"
	"time"
)

func TestFirstNonEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{"all empty", []string{"", "   "}, ""},
		{"first non empty", []string{"foo", "bar"}, "foo"},
		{"skips whitespace", []string{"   ", "bar"}, "bar"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := firstNonEmpty(tt.values...); got != tt.want {
				t.Fatalf("firstNonEmpty(%v) = %q, want %q", tt.values, got, tt.want)
			}
		})
	}
}

func TestParseIntWithDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		def   int
		want  int
	}{
		{"blank returns default", "", 7, 7},
		{"invalid returns default", "abc", 3, 3},
		{"valid parses value", "42", 0, 42},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := parseIntWithDefault(tt.value, tt.def); got != tt.want {
				t.Fatalf("parseIntWithDefault(%q, %d) = %d, want %d", tt.value, tt.def, got, tt.want)
			}
		})
	}
}

func TestParseDurationWithDefault(t *testing.T) {
	t.Parallel()

	def := 5 * time.Second
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{"blank returns default", "", def},
		{"invalid returns default", "nonsense", def},
		{"valid parses", "2m", 2 * time.Minute},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := parseDurationWithDefault(tt.value, def); got != tt.want {
				t.Fatalf("parseDurationWithDefault(%q) = %s, want %s", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseBoolWithDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		def   bool
		want  bool
	}{
		{"blank returns default", "", true, true},
		{"invalid returns default", "nope", false, false},
		{"valid parses", "true", false, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := parseBoolWithDefault(tt.value, tt.def); got != tt.want {
				t.Fatalf("parseBoolWithDefault(%q, %t) = %t, want %t", tt.value, tt.def, got, tt.want)
			}
		})
	}
}

func TestLoadUsesEnvironmentDefaults(t *testing.T) {
	t.Setenv("SERVER_ADDR", "")
	t.Setenv("ADDR", "")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("DATABASE_MAX_IDLE_CONNS", "10")
	t.Setenv("DATABASE_MAX_OPEN_CONNS", "100")
	t.Setenv("DATABASE_CONN_MAX_LIFETIME", "1h")
	t.Setenv("DATABASE_CONN_MAX_IDLE_TIME", "30m")
	t.Setenv("DATABASE_USE_MOCK", "true")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SESSION_LIFETIME", "45m")
	t.Setenv("SESSION_COOKIE_NAME", "custom_session")
	t.Setenv("SESSION_COOKIE_DOMAIN", "example.com")
	t.Setenv("SESSION_COOKIE_SECURE", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Addr != ":8080" {
		t.Fatalf("Server.Addr = %q, want %q", cfg.Server.Addr, ":8080")
	}
	if cfg.Database.URL != "postgres://example" {
		t.Fatalf("Database.URL = %q", cfg.Database.URL)
	}
	if cfg.Database.MaxIdleConns != 10 {
		t.Fatalf("Database.MaxIdleConns = %d", cfg.Database.MaxIdleConns)
	}
	if cfg.Database.MaxOpenConns != 100 {
		t.Fatalf("Database.MaxOpenConns = %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.ConnMaxLifetime != time.Hour {
		t.Fatalf("Database.ConnMaxLifetime = %s", cfg.Database.ConnMaxLifetime)
	}
	if cfg.Database.ConnMaxIdleTime != 30*time.Minute {
		t.Fatalf("Database.ConnMaxIdleTime = %s", cfg.Database.ConnMaxIdleTime)
	}
	if !cfg.Database.UseMock {
		t.Fatalf("Database.UseMock = %t, want true", cfg.Database.UseMock)
	}
	if cfg.Logging.Level != "debug" {
		t.Fatalf("Logging.Level = %q", cfg.Logging.Level)
	}
	if cfg.Auth.Session.Lifetime != 45*time.Minute {
		t.Fatalf("Auth.Session.Lifetime = %s", cfg.Auth.Session.Lifetime)
	}
	if cfg.Auth.Session.CookieName != "custom_session" {
		t.Fatalf("Auth.Session.CookieName = %q", cfg.Auth.Session.CookieName)
	}
	if cfg.Auth.Session.CookieDomain != "example.com" {
		t.Fatalf("Auth.Session.CookieDomain = %q", cfg.Auth.Session.CookieDomain)
	}
	if cfg.Auth.Session.CookieSecure {
		t.Fatalf("Auth.Session.CookieSecure = %t, want false", cfg.Auth.Session.CookieSecure)
	}
}

func TestLoadPrefersServerAddr(t *testing.T) {
	t.Setenv("SERVER_ADDR", "127.0.0.1:9000")
	t.Setenv("DATABASE_URL", "postgres://example")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Addr != "127.0.0.1:9000" {
		t.Fatalf("Server.Addr = %q, want %q", cfg.Server.Addr, "127.0.0.1:9000")
	}
}
