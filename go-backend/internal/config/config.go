package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
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

// DSNForLog безопасный вывод DSN без пароля для логирования
func (p *Config) DSNForLog() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=*** dbname=%s sslmode=%s",
		p.DBHost, p.DBPort, p.DBUser, p.DBName, p.DBSSLMode)
}

func (c *Config) IsDev() bool {
	return c.Environment == "dev"
}

func LoadConfig() *Config {
	// Загрузка .env файла (если существует)
	if err := godotenv.Load(); err != nil {
		// Игнорируем ошибку, если файл не найден - используем переменные окружения системы
		log.Println("No .env file found, using system environment variables")
	}

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
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           getEnv("DB_USER", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", ""),
		DBName:           getEnv("DB_NAME", "ai_detector"),
		DBSSLMode:        getEnv("DB_SSLMODE", "disable"),
	}

	// Проверка обязательных полей
	if cfg.DBPassword == "" {
		fmt.Println("WARNING: DB_PASSWORD is not set!")
	}
	if cfg.DBName == "" {
		fmt.Println("WARNING: DB_NAME is not set, using default: ai_detector")
		cfg.DBName = "ai_detector"
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
