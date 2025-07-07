package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramBotToken      string
	ServerPort            string
	LogLevel              string
	RequestTimeout        int
	MaxConcurrentRequests int
	StorageAdapter        string // "inmemory", "postgres", "sqlite"
	PostgresDSN           string // Data Source Name for PostgreSQL
	SQLiteDSN             string // Data Source Name for SQLite (e.g., "loopgate.db" or "file::memory:?cache=shared")
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		TelegramBotToken:      getEnv("TELEGRAM_BOT_TOKEN", ""),
		ServerPort:            getEnv("SERVER_PORT", "8080"),
		LogLevel:              getEnv("LOG_LEVEL", "info"),
		RequestTimeout:        getEnvInt("REQUEST_TIMEOUT", 300),
		MaxConcurrentRequests: getEnvInt("MAX_CONCURRENT_REQUESTS", 100),
		StorageAdapter:        getEnv("STORAGE_ADAPTER", "postgres"), // Default to SQLite
		PostgresDSN:           getEnv("POSTGRES_DSN", "host=localhost user=postgres password=postgres dbname=loopgate port=5432 sslmode=disable"),          // e.g., "host=localhost user=user password=pass dbname=loopgate port=5432 sslmode=disable"
		SQLiteDSN:             getEnv("SQLITE_DSN", "loopgate.db"), // Default to a local file "loopgate.db"
	}

	// Validate storage adapter choice
	switch cfg.StorageAdapter {
	case "inmemory", "postgres", "sqlite":
		// valid
	default:
		log.Fatalf("Invalid STORAGE_ADAPTER: %s. Must be one of 'inmemory', 'postgres', 'sqlite'", cfg.StorageAdapter)
	}

	if cfg.StorageAdapter == "postgres" && cfg.PostgresDSN == "" {
		log.Fatalf("POSTGRES_DSN must be set when STORAGE_ADAPTER is 'postgres'")
	}
	if cfg.StorageAdapter == "sqlite" && cfg.SQLiteDSN == "" {
		log.Fatalf("SQLITE_DSN must be set when STORAGE_ADAPTER is 'sqlite'")
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