package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	applog "perfugo/internal/log"
)

// Config captures the runtime configuration for the application.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logging  LoggingConfig
	Auth     AuthConfig
	AI       AIConfig
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
	UseMock         bool
}

// LoggingConfig controls application logging behavior.
type LoggingConfig struct {
	Level string
}

// AuthConfig controls authentication and session behavior for the application.
type AuthConfig struct {
	Session SessionConfig
}

// AIConfig controls OpenAI integration behaviour.
type AIConfig struct {
	APIKey         string
	Model          string
	BaseURL        string
	RequestTimeout time.Duration
}

// SessionConfig configures HTTP session cookie behavior.
type SessionConfig struct {
	Lifetime     time.Duration
	CookieName   string
	CookieDomain string
	CookieSecure bool
}

// Load inspects the environment and builds a Config value.
func Load() (Config, error) {
	applog.Debug(context.Background(), "loading configuration from environment")
	cfg := Config{}

	cfg.Server = ServerConfig{
		Addr: firstNonEmpty(
			os.Getenv("SERVER_ADDR"),
			os.Getenv("ADDR"),
			":8080",
		),
	}

	applog.Debug(context.Background(), "server configuration resolved", "addr", cfg.Server.Addr)

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
		UseMock:         parseBoolWithDefault(os.Getenv("DATABASE_USE_MOCK"), false),
	}

	applog.Debug(context.Background(), "database configuration resolved",
		"urlConfigured", strings.TrimSpace(cfg.Database.URL) != "",
		"maxIdleConns", cfg.Database.MaxIdleConns,
		"maxOpenConns", cfg.Database.MaxOpenConns,
		"useMock", cfg.Database.UseMock,
	)

	cfg.Logging = LoggingConfig{
		Level: firstNonEmpty(
			os.Getenv("LOG_LEVEL"),
			"info",
		),
	}

	applog.Debug(context.Background(), "logging configuration resolved", "level", cfg.Logging.Level)

	cfg.Auth = AuthConfig{
		Session: SessionConfig{
			Lifetime:     parseDurationWithDefault(os.Getenv("SESSION_LIFETIME"), 12*time.Hour),
			CookieName:   firstNonEmpty(os.Getenv("SESSION_COOKIE_NAME"), "perfugo_session"),
			CookieDomain: os.Getenv("SESSION_COOKIE_DOMAIN"),
			CookieSecure: parseBoolWithDefault(os.Getenv("SESSION_COOKIE_SECURE"), true),
		},
	}

	applog.Debug(context.Background(), "session configuration resolved",
		"lifetime", cfg.Auth.Session.Lifetime.String(),
		"cookieName", cfg.Auth.Session.CookieName,
		"cookieDomainSet", strings.TrimSpace(cfg.Auth.Session.CookieDomain) != "",
		"cookieSecure", cfg.Auth.Session.CookieSecure,
	)

	cfg.AI = AIConfig{
		APIKey:         strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		Model:          firstNonEmpty(os.Getenv("OPENAI_MODEL"), defaultAIModel()),
		BaseURL:        strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")),
		RequestTimeout: parseDurationWithDefault(os.Getenv("OPENAI_TIMEOUT"), 90*time.Second),
	}

	applog.Debug(context.Background(), "ai configuration resolved",
		"apiKeySet", cfg.AI.APIKey != "",
		"model", cfg.AI.Model,
		"baseURL", cfg.AI.BaseURL,
		"timeout", cfg.AI.RequestTimeout.String(),
	)

	if strings.TrimSpace(cfg.Server.Addr) == "" {
		return Config{}, fmt.Errorf("server address must not be empty")
	}

	applog.Debug(context.Background(), "configuration load complete")

	return cfg, nil
}

func defaultAIModel() string {
	return "gpt-4.1-mini"
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
