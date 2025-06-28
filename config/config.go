package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	TelegramBotToken string
	ServerPort       int
	LogLevel         string
}

func Load() (*Config, error) {
	cfg := &Config{
		ServerPort: 8080,
		LogLevel:   "info",
	}

	if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
		cfg.TelegramBotToken = token
	} else {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.ServerPort = p
		}
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.LogLevel = level
	}

	return cfg, nil
}