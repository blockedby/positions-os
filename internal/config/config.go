// package config loads application configuration from environment variables.
package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration.
type Config struct {
	// database
	DatabaseURL string

	// nats
	NatsURL string

	// llm
	LLMBaseURL string
	LLMModel   string

	// telegram
	TGApiID      int
	TGApiHash    string
	TGSessionStr string

	// server
	HTTPPort int

	// logging
	LogLevel string
	LogFile  string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://jhos:jhos_secret@localhost:5432/jhos?sslmode=disable"),
		NatsURL:      getEnv("NATS_URL", "nats://localhost:4222"),
		LLMBaseURL:   getEnv("LLM_BASE_URL", "http://localhost:1234/v1"),
		LLMModel:     getEnv("LLM_MODEL", "local-model"),
		TGApiHash:    getEnv("TG_API_HASH", ""),
		TGSessionStr: getEnv("TG_SESSION_STRING", ""),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		LogFile:      getEnv("LOG_FILE", "./logs/app.log"),
		HTTPPort:     getEnvInt("HTTP_PORT", 3100),
		TGApiID:      getEnvInt("TG_API_ID", 0),
	}

	return cfg, nil
}

// getEnv returns the value of an environment variable or a default value.
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvInt returns the integer value of an environment variable or a default.
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
