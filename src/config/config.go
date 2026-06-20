package config

import (
	"fmt"
	"os"
	"strconv"
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
	// RAG (E05)
	EmbeddingsDriver string // "voyage" | "stub"
	VoyageKey        string
	KnowledgeMaxDist float64 // umbral de distancia coseno (P-05)
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

		EmbeddingsDriver: env("EMBEDDINGS_DRIVER", "voyage"),
		VoyageKey:        os.Getenv("VOYAGE_API_KEY"),
		KnowledgeMaxDist: envFloat("KNOWLEDGE_MAX_DISTANCE", 0.85),
	}
}

func envFloat(k string, def float64) float64 {
	if v := os.Getenv(k); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

// DSN con search_path=river,public: River usa el schema river (primero); public
// queda disponible para el tipo `vector` (extensión pgvector) y pgcrypto. Las
// consultas a core.tenants / knowledge van calificadas por schema.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&search_path=river,public",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
