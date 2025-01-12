package config

import (
	"context"
	"time"

	"github.com/sethvargo/go-envconfig"
)

// Config holds the configuration for the Inference Gateway.
type Config struct {
	// General settings
	ApplicationName string `env:"APPLICATION_NAME, default=inference-gateway"`
	EnableTelemetry bool   `env:"ENABLE_TELEMETRY, default=false"`
	Environment     string `env:"ENVIRONMENT, default=production"`

	// Server settings
	ServerHost         string        `env:"SERVER_HOST, default=127.0.0.1"`
	ServerPort         string        `env:"SERVER_PORT, default=8080"`
	ServerReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT, default=30s"`
	ServerWriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT, default=30s"`
	ServerIdleTimeout  time.Duration `env:"SERVER_IDLE_TIMEOUT, default=120s"`
	ServerTLSCertPath  string        `env:"SERVER_TLS_CERT_PATH"`
	ServerTLSKeyPath   string        `env:"SERVER_TLS_KEY_PATH"`

	// API URLs and keys
	OllamaAPIURL      string `env:"OLLAMA_API_URL, default=http://ollama:8080"`
	GroqAPIURL        string `env:"GROQ_API_URL, default=https://api.groq.com"`
	GroqAPIKey        string `env:"GROQ_API_KEY"`
	OpenaiAPIURL      string `env:"OPENAI_API_URL, default=https://api.openai.com"`
	OpenaiAPIKey      string `env:"OPENAI_API_KEY"`
	GoogleAIStudioURL string `env:"GOOGLE_AISTUDIO_API_URL, default=https://generativelanguage.googleapis.com"`
	GoogleAIStudioKey string `env:"GOOGLE_AISTUDIO_API_KEY"`
}

// Load loads the configuration from environment variables.
func (cfg *Config) Load() (Config, error) {
	if err := envconfig.Process(context.Background(), cfg); err != nil {
		return Config{}, err
	}
	return *cfg, nil
}
