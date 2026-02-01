package config

import (
	"fmt"
	"os"
)

// Config holds all application configuration
type Config struct {
	Version string
	Port    string
	
	// Database
	DatabaseURL string
	
	// Redis
	RedisURL string
	
	// Kafka/Redpanda
	KafkaBrokers string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Version:      getEnv("ADE_VERSION", "0.1.0"),
		Port:         getEnv("ADE_PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://ade:ade@localhost:5432/ade?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379/0"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
	}

	if cfg.Port == "" {
		return nil, fmt.Errorf("port cannot be empty")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
