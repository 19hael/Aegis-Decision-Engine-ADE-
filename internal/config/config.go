package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	Server ServerConfig
	
	// Database configuration
	Database DatabaseConfig
	
	// Redis configuration
	Redis RedisConfig
	
	// Kafka configuration
	Kafka KafkaConfig
	
	// Feature configuration
	Features FeatureConfig
	
	// Simulation configuration
	Simulation SimulationConfig
	
	// Action configuration
	Action ActionConfig
	
	// Logging configuration
	Logging LoggingConfig
	
	// Metrics configuration
	Metrics MetricsConfig
	
	// Rate limiting configuration
	RateLimit RateLimitConfig
	
	// Application metadata
	Version string
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port           string
	Host           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	URL             string
	MaxConnections  int32
	MinConnections  int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

// KafkaConfig holds Kafka/Redpanda configuration
type KafkaConfig struct {
	Brokers       []string
	EventsTopic   string
	ActionsTopic  string
	ConsumerGroup string
}

// FeatureConfig holds feature calculation configuration
type FeatureConfig struct {
	WindowSize  time.Duration
	SnapshotTTL time.Duration
}

// SimulationConfig holds simulation configuration
type SimulationConfig struct {
	DefaultIterations int
	MaxIterations     int
	DefaultHorizon    time.Duration
	MaxHorizon        time.Duration
}

// ActionConfig holds action execution configuration
type ActionConfig struct {
	DefaultWebhookTimeout time.Duration
	MaxRetries            int
	EnableCircuitBreaker  bool
	CircuitBreaker        CircuitBreakerConfig
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	MaxFailures  int
	ResetTimeout time.Duration
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Format string
	Output string
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled bool
	Path    string
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled bool
	Rate    int
	Burst   int
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Version: getEnv("ADE_VERSION", "1.0.0"),
		
		Server: ServerConfig{
			Port:            getEnv("ADE_PORT", "8080"),
			Host:            getEnv("ADE_HOST", "0.0.0.0"),
			ReadTimeout:     parseDuration("ADE_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    parseDuration("ADE_WRITE_TIMEOUT", 10*time.Second),
			ShutdownTimeout: parseDuration("ADE_SHUTDOWN_TIMEOUT", 5*time.Second),
		},
		
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://ade:ade@localhost:5432/ade?sslmode=disable"),
			MaxConnections:  int32(parseInt("DB_MAX_CONNECTIONS", 20)),
			MinConnections:  int32(parseInt("DB_MIN_CONNECTIONS", 5)),
			MaxConnLifetime: parseDuration("DB_MAX_CONN_LIFETIME", time.Hour),
			MaxConnIdleTime: parseDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		},
		
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6379/0"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       parseInt("REDIS_DB", 0),
		},
		
		Kafka: KafkaConfig{
			Brokers:       parseStringSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			EventsTopic:   getEnv("KAFKA_EVENTS_TOPIC", "ade.events"),
			ActionsTopic:  getEnv("KAFKA_ACTIONS_TOPIC", "ade.actions"),
			ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "ade-consumer"),
		},
		
		Features: FeatureConfig{
			WindowSize:  parseDuration("FEATURE_WINDOW_SIZE", 5*time.Minute),
			SnapshotTTL: parseDuration("FEATURE_SNAPSHOT_TTL", 10*time.Minute),
		},
		
		Simulation: SimulationConfig{
			DefaultIterations: parseInt("SIMULATION_DEFAULT_ITERATIONS", 1000),
			MaxIterations:     parseInt("SIMULATION_MAX_ITERATIONS", 10000),
			DefaultHorizon:    parseDuration("SIMULATION_DEFAULT_HORIZON", 10*time.Minute),
			MaxHorizon:        parseDuration("SIMULATION_MAX_HORIZON", 15*time.Minute),
		},
		
		Action: ActionConfig{
			DefaultWebhookTimeout: parseDuration("ACTION_WEBHOOK_TIMEOUT", 30*time.Second),
			MaxRetries:            parseInt("ACTION_MAX_RETRIES", 3),
			EnableCircuitBreaker:  parseBool("ACTION_ENABLE_CIRCUIT_BREAKER", true),
			CircuitBreaker: CircuitBreakerConfig{
				MaxFailures:  parseInt("CB_MAX_FAILURES", 5),
				ResetTimeout: parseDuration("CB_RESET_TIMEOUT", 30*time.Second),
			},
		},
		
		Logging: LoggingConfig{
			Level:  getEnv("ADE_LOG_LEVEL", "info"),
			Format: getEnv("ADE_LOG_FORMAT", "json"),
			Output: getEnv("ADE_LOG_OUTPUT", "stdout"),
		},
		
		Metrics: MetricsConfig{
			Enabled: parseBool("METRICS_ENABLED", true),
			Path:    getEnv("METRICS_PATH", "/metrics"),
		},
		
		RateLimit: RateLimitConfig{
			Enabled: parseBool("RATE_LIMIT_ENABLED", true),
			Rate:    parseInt("RATE_LIMIT_RATE", 100),
			Burst:   parseInt("RATE_LIMIT_BURST", 200),
		},
	}
	
	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func parseBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func parseDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func parseStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple parsing - in production use proper CSV parsing
		return []string{value}
	}
	return defaultValue
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}
	if c.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("at least one Kafka broker is required")
	}
	return nil
}
