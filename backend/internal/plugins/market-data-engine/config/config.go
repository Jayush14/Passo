package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	LLM        LLMConfig
	Processing ProcessingConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         string
	Environment  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// LLMConfig holds LLM provider configuration
type LLMConfig struct {
	DefaultProvider string
	OpenAI          OpenAIConfig
	Anthropic       AnthropicConfig
	Google          GoogleConfig
}

// OpenAIConfig holds OpenAI-specific configuration
type OpenAIConfig struct {
	APIKey string
	Model  string
	// Temperature, max tokens, etc. can be added as needed
}

// AnthropicConfig holds Anthropic-specific configuration
type AnthropicConfig struct {
	APIKey string
	Model  string
}

// GoogleConfig holds Google AI-specific configuration
type GoogleConfig struct {
	APIKey string
	Model  string
}

// ProcessingConfig holds job processing configuration
type ProcessingConfig struct {
	MaxConcurrentJobs   int
	JobTimeoutSeconds   int
	MaxRetryAttempts    int
	BaseRetryDelay      time.Duration
	MaxRetryDelay       time.Duration
	ConfidenceThreshold float64
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Environment:  getEnv("SERVER_ENV", "development"),
			ReadTimeout:  time.Duration(getEnvAsInt("SERVER_READ_TIMEOUT", 120)) * time.Second,
			WriteTimeout: time.Duration(getEnvAsInt("SERVER_WRITE_TIMEOUT", 120)) * time.Second,
			IdleTimeout:  time.Duration(getEnvAsInt("SERVER_IDLE_TIMEOUT", 120)) * time.Second,
		},
		Database: DatabaseConfig{
			Host:            getEnv("POSTGRES_HOST", "localhost"),
			Port:            getEnv("POSTGRES_PORT", "5432"),
			User:            getEnv("POSTGRES_USER", "postgres"),
			Password:        getEnv("POSTGRES_PASSWORD", "password"),
			DBName:          getEnv("POSTGRES_DB", "market_data"),
			SSLMode:         getEnv("POSTGRES_SSLMODE", "disable"),
			MaxOpenConns:    getEnvAsInt("POSTGRES_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("POSTGRES_MAX_IDLE_CONNS", 25),
			ConnMaxLifetime: time.Duration(getEnvAsInt("POSTGRES_CONN_MAX_LIFETIME", 300)) * time.Second,
		},
		LLM: LLMConfig{
			DefaultProvider: getEnv("LLM_DEFAULT_PROVIDER", "mock"),
			OpenAI: OpenAIConfig{
				APIKey: getEnv("OPENAI_API_KEY", ""),
				Model:  getEnv("OPENAI_MODEL", "gpt-4-turbo-preview"),
			},
			Anthropic: AnthropicConfig{
				APIKey: getEnv("ANTHROPIC_API_KEY", ""),
				Model:  getEnv("ANTHROPIC_MODEL", "claude-3-opus-20240229"),
			},
			Google: GoogleConfig{
				APIKey: getEnv("GOOGLE_API_KEY", ""),
				Model:  getEnv("GOOGLE_MODEL", "gemini-pro"),
			},
		},
		Processing: ProcessingConfig{
			MaxConcurrentJobs:   getEnvAsInt("MAX_CONCURRENT_JOBS", 10),
			JobTimeoutSeconds:   getEnvAsInt("JOB_TIMEOUT_SECONDS", 300),
			MaxRetryAttempts:    getEnvAsInt("MAX_RETRY_ATTEMPTS", 3),
			BaseRetryDelay:      time.Duration(getEnvAsInt("BASE_RETRY_DELAY_SECONDS", 1)) * time.Second,
			MaxRetryDelay:       time.Duration(getEnvAsInt("MAX_RETRY_DELAY_SECONDS", 60)) * time.Second,
			ConfidenceThreshold: getEnvAsFloat("CONFIDENCE_THRESHOLD", 0.7),
		},
	}

	// Validate required configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("POSTGRES_HOST is required")
	}
	if c.Database.Port == "" {
		return fmt.Errorf("POSTGRES_PORT is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("POSTGRES_USER is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("POSTGRES_DB is required")
	}
	if c.Server.Port == "" {
		return fmt.Errorf("SERVER_PORT is required")
	}
	return nil
}

// Helper functions to get environment variables with fallbacks
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if val, err := fmt.Sscanf(value, "%d", &defaultValue); err == nil && val == 1 {
			return defaultValue
		}
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if val, err := fmt.Sscanf(value, "%f", &defaultValue); err == nil && val == 1 {
			return defaultValue
		}
	}
	return defaultValue
}
