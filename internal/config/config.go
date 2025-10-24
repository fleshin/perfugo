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
