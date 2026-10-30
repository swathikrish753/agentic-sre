package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config is the top-level application configuration, assembled from
// environment variables with sensible defaults so the project runs
// locally with zero setup.
type Config struct {
	Log LogConfig
	LLM LLMConfig
}

// LogConfig controls structured logging behaviour.
type LogConfig struct {
	Level  string // debug | info | warn | error
	Format string // json | text
}

// LLMConfig points the agent at a model provider (local Ollama for now).
type LLMConfig struct {
	Provider string        // "ollama"
	BaseURL  string        // e.g. http://localhost:11434
	Model    string        // model tag pulled in Ollama
	Timeout  time.Duration // per-request timeout
}

// Load reads config from the environment, applies defaults, and validates.
func Load() (Config, error) {
	cfg := Config{
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "text"),
		},
		LLM: LLMConfig{
			Provider: getEnv("LLM_PROVIDER", "ollama"),
			BaseURL:  getEnv("LLM_BASE_URL", "http://localhost:11434"),
			Model:    getEnv("LLM_MODEL", "llama3.2:3b"),
			Timeout:  getEnvDuration("LLM_TIMEOUT", 60*time.Second),
		},
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if !oneOf(c.Log.Level, "debug", "info", "warn", "error") {
		return fmt.Errorf("config: invalid LOG_LEVEL %q", c.Log.Level)
	}
	if !oneOf(c.Log.Format, "json", "text") {
		return fmt.Errorf("config: invalid LOG_FORMAT %q", c.Log.Format)
	}
	if c.LLM.Model == "" {
		return fmt.Errorf("config: LLM_MODEL must not be empty")
	}
	if c.LLM.Timeout <= 0 {
		return fmt.Errorf("config: LLM_TIMEOUT must be positive")
	}
	return nil
}

// --- small env helpers (stdlib only) ---

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func oneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if strings.EqualFold(v, a) {
			return true
		}
	}
	return false
}
