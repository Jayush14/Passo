package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds application configuration.
type Config struct {
	Port              string
	Env               string
	DBURL             string
	JWTSecret         string
	TokenExpiry       time.Duration
	EncryptionKey     string // 32-byte key for AES-256-GCM
	HouseKey          string // optional house key for subscription tier
	VAPIDPublicKey    string
	VAPIDPrivateKey   string
	GeminiAPIEndpoint string // Gemini API endpoint
	GeminiModel       string // Gemini model to use
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() *Config {
	return &Config{
		Port:            getEnv("PORT", "8080"),
		Env:             getEnv("ENV", "development"),
		DBURL:           getEnv("DB_URL", "postgres://postgres:postgres@localhost:5432/passo?sslmode=disable"),
		JWTSecret:       getEnv("JWT_SECRET", "supersecret"),
		TokenExpiry:     time.Duration(getEnvAsInt("TOKEN_EXPIRY_HOURS", 24)) * time.Hour,
		EncryptionKey:   getEnv("ENCRYPTION_KEY", ""), // must be set in prod
		HouseKey:        getEnv("HOUSE_KEY", ""),
		VAPIDPublicKey:  getEnv("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey: getEnv("VAPID_PRIVATE_KEY", ""),
		// GeminiAPIEndpoint: getEnv("GEMINI_API_ENDPOINT", "https://generativelanguage.googleapis.com/v1beta/models"),
		// GeminiModel:       getEnv("GEMINI_MODEL", "gemini-3.5-flash-lite"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i := parseInt(v); i != nil {
			return *i
		}
	}
	return fallback
}

func parseInt(s string) *int {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	if err != nil {
		return nil
	}
	return &i
}
