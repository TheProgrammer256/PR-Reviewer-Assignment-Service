package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPPort  int
	DBHost    string
	DBPort    int
	DBUser    string
	DBPass    string
	DBName    string
	DBSSLMode string
}

func Load() Config {
	return Config{
		HTTPPort:  intFromEnv("HTTP_PORT", 8080),
		DBHost:    strFromEnv("DB_HOST", "db"),
		DBPort:    intFromEnv("DB_PORT", 5432),
		DBUser:    strFromEnv("DB_USER", "app"),
		DBPass:    strFromEnv("DB_PASSWORD", "app"),
		DBName:    strFromEnv("DB_NAME", "pr_assignments"),
		DBSSLMode: strFromEnv("DB_SSLMODE", "disable"),
	}
}

func (c Config) Addr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}

func (c Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DBUser,
		c.DBPass,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}

func strFromEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func intFromEnv(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return fallback
}
