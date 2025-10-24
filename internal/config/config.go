package config

import (
	"fmt"
	"os"
	"strings"
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
	URL string
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
