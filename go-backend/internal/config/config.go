package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	GRPCPort         string
	HTTPPort         string
	PythonServiceURL string
	CORSOrigins      string
	MaxConnections   int
	RateLimitPerMin  int
	MaxMessageSizeMB int
	LogLevel         string
	Environment      string
}

func LoadConfig() *Config {
	cfg := &Config{
		GRPCPort:         getEnv("GRPC_PORT", "50051"),
		HTTPPort:         getEnv("HTTP_PORT", "8080"),
		PythonServiceURL: getEnv("PYTHON_SERVICE_URL", "localhost:9000"),
		CORSOrigins:      getEnv("CORS_ORIGINS", "http://localhost:3000"),
		MaxConnections:   getEnvInt("MAX_CONNECTIONS", 1000),
		RateLimitPerMin:  getEnvInt("RATE_LIMIT_PER_MIN", 100),
		MaxMessageSizeMB: getEnvInt("MAX_MESSAGE_SIZE_MB", 50),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		Environment:      getEnv("ENVIRONMENT", "dev"),
	}

	log.Println("Configuration loaded:")
	log.Printf("GRPC Port: %s", cfg.GRPCPort)
	log.Printf("Python Service: %s", cfg.PythonServiceURL)

	return cfg
}

func getEnv(key string, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func (c *Config) IsDev() bool {
	return c.Environment == "dev"
}
