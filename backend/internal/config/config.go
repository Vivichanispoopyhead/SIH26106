package config

import (
	"os"
	"strings"
)

// Config contains environment-specific application settings.
type Config struct {
	Address        string
	Environment    string
	AllowedOrigins []string
}

// Load reads configuration from the process environment and applies safe local defaults.
func Load() Config {
	return Config{
		Address:        valueOrDefault("APP_ADDR", ":8080"),
		Environment:    valueOrDefault("APP_ENV", "development"),
		AllowedOrigins: commaSeparated("APP_ALLOWED_ORIGINS", "http://localhost:5173"),
	}
}

func valueOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return fallback
}

func commaSeparated(key, fallback string) []string {
	values := strings.Split(valueOrDefault(key, fallback), ",")
	origins := make([]string, 0, len(values))
	for _, value := range values {
		if origin := strings.TrimSpace(value); origin != "" {
			origins = append(origins, origin)
		}
	}

	return origins
}
