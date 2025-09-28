package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	Port        string
	Environment string

	// Database configuration (Supabase PostgreSQL)
	PostgresURL    string
	SupabaseConfig SupabaseConfig

	// Storage configuration (Google Cloud Storage)
	StorageConfig StorageConfig

	// Event publishing configuration (Confluent Cloud Kafka)
	KafkaConfig KafkaConfig

	// Authentication configuration
	JWTSecret string
	JWTExpiry string

	// Feature flags
	Features FeatureConfig

	// Monitoring and observability
	Monitoring MonitoringConfig
}

// SupabaseConfig holds Supabase-specific configuration
type SupabaseConfig struct {
	URL            string
	AnonKey        string
	ServiceRoleKey string
	JWTSecret      string
}

// StorageConfig holds Google Cloud Storage configuration
type StorageConfig struct {
	ProjectID       string
	BucketName      string
	CredentialsPath string
	CredentialsJSON string // For containerized environments
	CDNDomain       string
}

// KafkaConfig holds Confluent Cloud Kafka configuration
type KafkaConfig struct {
	BootstrapServers string
	APIKey           string
	APISecret        string
	Topic            string
	BatchSize        int
	BatchTimeout     string
	Retries          int
	Acks             string
}

// FeatureConfig holds feature flags
type FeatureConfig struct {
	AnalyticsEnabled           bool
	RealTimeUpdatesEnabled     bool
	ImageOptimizationEnabled   bool
	ThumbnailGenerationEnabled bool
}

// MonitoringConfig holds monitoring and observability configuration
type MonitoringConfig struct {
	LogLevel           string
	LogFormat          string
	MetricsEnabled     bool
	MetricsPort        string
	HealthCheckEnabled bool
	TracingEnabled     bool
	TraceSampleRate    float64
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),

		// Database configuration (Supabase PostgreSQL)
		PostgresURL: getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/posts_db?sslmode=disable"),
		SupabaseConfig: SupabaseConfig{
			URL:            getEnv("SUPABASE_URL", ""),
			AnonKey:        getEnv("SUPABASE_ANON_KEY", ""),
			ServiceRoleKey: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
			JWTSecret:      getEnv("SUPABASE_JWT_SECRET", ""),
		},

		// Storage configuration (Google Cloud Storage)
		StorageConfig: StorageConfig{
			ProjectID:       getEnv("BUCKET_PROJECT_ID", ""),
			BucketName:      getEnv("BUCKET_NAME", "posts-bucket"),
			CredentialsPath: getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
			CredentialsJSON: getEnv("BUCKET_SERVICE_ACCOUNT_KEY", ""),
			CDNDomain:       getEnv("BUCKET_CDN_DOMAIN", ""),
		},

		// Event publishing configuration (Confluent Cloud Kafka)
		KafkaConfig: KafkaConfig{
			BootstrapServers: getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
			APIKey:           getEnv("KAFKA_API_KEY", ""),
			APISecret:        getEnv("KAFKA_API_SECRET", ""),
			Topic:            getEnv("KAFKA_TOPIC", "posts.events"),
			BatchSize:        getIntEnv("KAFKA_BATCH_SIZE", 100),
			BatchTimeout:     getEnv("KAFKA_BATCH_TIMEOUT", "10ms"),
			Retries:          getIntEnv("KAFKA_RETRIES", 3),
			Acks:             getEnv("KAFKA_ACKS", "1"),
		},

		// Authentication configuration
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key"),
		JWTExpiry: getEnv("JWT_EXPIRY", "24h"),

		// Feature flags
		Features: FeatureConfig{
			AnalyticsEnabled:           getBoolEnv("FEATURE_ANALYTICS_ENABLED", true),
			RealTimeUpdatesEnabled:     getBoolEnv("FEATURE_REAL_TIME_UPDATES", true),
			ImageOptimizationEnabled:   getBoolEnv("FEATURE_IMAGE_OPTIMIZATION", true),
			ThumbnailGenerationEnabled: getBoolEnv("FEATURE_THUMBNAIL_GENERATION", true),
		},

		// Monitoring configuration
		Monitoring: MonitoringConfig{
			LogLevel:           getEnv("LOG_LEVEL", "info"),
			LogFormat:          getEnv("LOG_FORMAT", "text"),
			MetricsEnabled:     getBoolEnv("METRICS_ENABLED", false),
			MetricsPort:        getEnv("METRICS_PORT", "9090"),
			HealthCheckEnabled: getBoolEnv("HEALTH_CHECK_ENABLED", true),
			TracingEnabled:     getBoolEnv("TRACING_ENABLED", false),
			TraceSampleRate:    getFloatEnv("TRACE_SAMPLE_RATE", 0.1),
		},
	}
}

// Utility functions for parsing environment variables

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}
