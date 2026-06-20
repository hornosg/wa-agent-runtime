package config

import (
	"fmt"
	"os"
)

// Config del agent-runtime (SVC-02). 12-factor (RULE-02).
type Config struct {
	DBHost       string
	DBPort       string
	DBName       string
	DBUser       string
	DBPassword   string
	DBSSLMode    string
	LLMDriver    string // "anthropic" (real) | "stub" (sin API key, dev/CI)
	AnthropicKey string
	MaxWorkers   int
}

func Load() Config {
	return Config{
		DBHost:       env("DB_HOST", "lab-postgres"),
		DBPort:       env("DB_PORT", "5432"),
		DBName:       env("DB_NAME", "whatsapp_agent"),
		DBUser:       env("DB_USER", "whatsapp_agent"),
		DBPassword:   env("DB_PASSWORD", "whatsapp_agent"),
		DBSSLMode:    env("DB_SSLMODE", "disable"),
		LLMDriver:    env("LLM_DRIVER", "anthropic"),
		AnthropicKey: os.Getenv("ANTHROPIC_API_KEY"),
		MaxWorkers:   5,
	}
}

// DSN con search_path=river: River usa el schema river; las consultas a core.tenants
// van calificadas (core.tenants), así que un solo pool sirve para ambos.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&search_path=river",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
