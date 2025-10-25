package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config captures the runtime configuration for the application.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logging  LoggingConfig
	Auth     AuthConfig
}

// ServerConfig configures the HTTP server runtime behavior.
type ServerConfig struct {
	Addr string
}

// DatabaseConfig contains the database connection settings.
type DatabaseConfig struct {
	URL             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// LoggingConfig controls application logging behavior.
type LoggingConfig struct {
	Level string
}

// AuthConfig controls authentication and session behavior for the application.
type AuthConfig struct {
	Session SessionConfig
	OIDC    OIDCConfig
}

// SessionConfig configures HTTP session cookie behavior.
type SessionConfig struct {
	Lifetime     time.Duration
	CookieName   string
	CookieDomain string
	CookieSecure bool
}

// OIDCConfig captures the OpenID Connect provider configuration used for login.
type OIDCConfig struct {
	ProviderName string
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// Load inspects the environment and builds a Config value.
func Load() (Config, error) {
	cfg := Config{}

	cfg.Server = ServerConfig{
		Addr: firstNonEmpty(
			os.Getenv("SERVER_ADDR"),
			os.Getenv("ADDR"),
			":8080",
		),
	}

	cfg.Database = DatabaseConfig{
		URL: firstNonEmpty(
			os.Getenv("DATABASE_URL"),
			os.Getenv("DB_URL"),
			"",
		),
		MaxIdleConns:    parseIntWithDefault(os.Getenv("DATABASE_MAX_IDLE_CONNS"), 5),
		MaxOpenConns:    parseIntWithDefault(os.Getenv("DATABASE_MAX_OPEN_CONNS"), 25),
		ConnMaxLifetime: parseDurationWithDefault(os.Getenv("DATABASE_CONN_MAX_LIFETIME"), 30*time.Minute),
		ConnMaxIdleTime: parseDurationWithDefault(os.Getenv("DATABASE_CONN_MAX_IDLE_TIME"), 5*time.Minute),
	}

	cfg.Logging = LoggingConfig{
		Level: firstNonEmpty(
			os.Getenv("LOG_LEVEL"),
			"info",
		),
	}

	cfg.Auth = AuthConfig{
		Session: SessionConfig{
			Lifetime:     parseDurationWithDefault(os.Getenv("SESSION_LIFETIME"), 12*time.Hour),
			CookieName:   firstNonEmpty(os.Getenv("SESSION_COOKIE_NAME"), "perfugo_session"),
			CookieDomain: os.Getenv("SESSION_COOKIE_DOMAIN"),
			CookieSecure: parseBoolWithDefault(os.Getenv("SESSION_COOKIE_SECURE"), true),
		},
		OIDC: OIDCConfig{
			ProviderName: firstNonEmpty(os.Getenv("OIDC_PROVIDER_NAME"), "google"),
			Issuer:       os.Getenv("OIDC_ISSUER"),
			ClientID:     os.Getenv("OIDC_CLIENT_ID"),
			ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("OIDC_REDIRECT_URL"),
			Scopes:       parseCommaSeparated(os.Getenv("OIDC_SCOPES"), []string{"profile", "email"}),
		},
	}

	if strings.TrimSpace(cfg.Server.Addr) == "" {
		return Config{}, fmt.Errorf("server address must not be empty")
	}

	return cfg, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parseIntWithDefault(value string, def int) int {
	if strings.TrimSpace(value) == "" {
		return def
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return def
	}
	return parsed
}

func parseDurationWithDefault(value string, def time.Duration) time.Duration {
	if strings.TrimSpace(value) == "" {
		return def
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return def
	}
	return parsed
}

func parseBoolWithDefault(value string, def bool) bool {
	if strings.TrimSpace(value) == "" {
		return def
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return def
	}
	return parsed
}

func parseCommaSeparated(value string, def []string) []string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' '
	})
	cleaned := make([]string, 0, len(fields))
	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	if len(cleaned) == 0 {
		return def
	}
	return cleaned
}
