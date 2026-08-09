package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv          string
	BaseURL         string
	HTTPAddr        string
	DatabaseURL     string
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	KafkaBrokers    []string
	KafkaClickTopic string
	OTLPEndpoint    string
	OTLPInsecure    bool
	ShutdownTimeout time.Duration
	CacheTTL        time.Duration
	DefaultLinkTTL  time.Duration
	FrontendOrigin  string
	ShortCodeLength int
}

func Load() Config {
	return Config{
		AppEnv:          getEnv("APP_ENV", "local"),
		BaseURL:         strings.TrimRight(getEnv("BASE_URL", "http://localhost:8080"), "/"),
		HTTPAddr:        getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://urlshorter:urlshorter@localhost:5432/urlshorter?sslmode=disable"),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		RedisDB:         getEnvInt("REDIS_DB", 0),
		KafkaBrokers:    splitCSV(getEnv("KAFKA_BROKERS", "localhost:9092")),
		KafkaClickTopic: getEnv("KAFKA_CLICK_TOPIC", "url-clicked"),
		OTLPEndpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		OTLPInsecure:    getEnvBool("OTEL_EXPORTER_OTLP_INSECURE", true),
		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		CacheTTL:        getEnvDuration("CACHE_TTL", 24*time.Hour),
		DefaultLinkTTL:  getEnvDuration("DEFAULT_LINK_TTL", 7*24*time.Hour),
		FrontendOrigin:  getEnv("FRONTEND_ORIGIN", "http://localhost:5173"),
		ShortCodeLength: getEnvInt("SHORT_CODE_LENGTH", 7),
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value, err := strconv.Atoi(getEnv(key, ""))
	if err != nil {
		return fallback
	}
	return value
}

func getEnvBool(key string, fallback bool) bool {
	value, err := strconv.ParseBool(getEnv(key, ""))
	if err != nil {
		return fallback
	}
	return value
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
