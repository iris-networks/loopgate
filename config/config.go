package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramBotToken      string
	ServerPort           string
	LogLevel             string
	RequestTimeout       int
	MaxConcurrentRequests int
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		TelegramBotToken:      getEnv("TELEGRAM_BOT_TOKEN", ""),
		ServerPort:           getEnv("SERVER_PORT", "8080"),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
		RequestTimeout:       getEnvInt("REQUEST_TIMEOUT", 300),
		MaxConcurrentRequests: getEnvInt("MAX_CONCURRENT_REQUESTS", 100),
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}