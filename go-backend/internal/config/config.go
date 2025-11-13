package config

import (
	"fmt"
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

	DBName     string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBSSLMode  string
}

func (p *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		p.DBHost, p.DBPort, p.DBUser, p.DBPassword, p.DBName, p.DBSSLMode)
}

func LoadConfig() *Config {
	cfg := &Config{
		GRPCPort:         getEnv("GRPC_PORT", "50051"),
		HTTPPort:         getEnv("HTTP_PORT", "8081"),
		PythonServiceURL: getEnv("PYTHON_SERVICE_URL", "http://python.service"),
		CORSOrigins:      getEnv("CORS_ORIGINS", "*"),
		MaxConnections:   getEnvInt("MAX_CONNECTIONS", 1000),
		RateLimitPerMin:  getEnvInt("RATE_PER_MIN", 1000),
		MaxMessageSizeMB: getEnvInt("MAX_MESSAGE_SIZE_MB", 50),
		LogLevel:         getEnv("LOG_LEVEL", "INFO"),
		Environment:      getEnv("ENVIRONMENT", "production"),
		DBHost:           getEnv("DB_HOST", "0.0.0.0"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           getEnv("DB_USER", ""),
		DBPassword:       getEnv("DB_PASSWORD", ""),
		DBName:           getEnv("DB_NAME", ""),
		DBSSLMode:        getEnv("DB_SSLMODE", "disable"),
	}

	return cfg
}

func getEnv(key string, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if intVal, err := strconv.Atoi(v); err == nil {
			return intVal
		}
	}
	return defaultVal
}
