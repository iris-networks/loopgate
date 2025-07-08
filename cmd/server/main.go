package main

import (
	"context"
	"log"
	"loopgate/config"
	"loopgate/internal/handlers"
	"loopgate/internal/mcp"
	"loopgate/internal/router"
	"loopgate/internal/session"
	"loopgate/internal/storage" // Added storage import
	"loopgate/internal/telegram"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := config.Load()

	if cfg.TelegramBotToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	// Initialize storage adapter based on configuration
	var storageAdapter storage.StorageAdapter
	var err error
	var closer func() // To store the Close function for database adapters

	switch cfg.StorageAdapter {
	case "inmemory":
		storageAdapter = storage.NewInMemoryStorageAdapter()
		log.Println("Using in-memory storage adapter")
	case "postgres":
		pgAdapter, pgErr := storage.NewPostgreSQLStorageAdapter(cfg.PostgresDSN)
		if pgErr != nil {
			log.Fatalf("Failed to initialize PostgreSQL storage adapter: %v", pgErr)
		}
		storageAdapter = pgAdapter
		closer = func() {
			log.Println("Closing PostgreSQL connection...")
			if err := pgAdapter.Close(); err != nil {
				log.Printf("Error closing PostgreSQL connection: %v", err)
			}
		}
		log.Println("Using PostgreSQL storage adapter")
	case "sqlite":
		sqliteAdapter, sqliteErr := storage.NewSQLiteStorageAdapter(cfg.SQLiteDSN)
		if sqliteErr != nil {
			log.Fatalf("Failed to initialize SQLite storage adapter: %v", sqliteErr)
		}
		storageAdapter = sqliteAdapter
		closer = func() {
			log.Println("Closing SQLite connection...")
			if err := sqliteAdapter.Close(); err != nil {
				log.Printf("Error closing SQLite connection: %v", err)
			}
		}
		log.Println("Using SQLite storage adapter")
	default:
		log.Fatalf("Invalid storage adapter configured: %s", cfg.StorageAdapter)
	}

	// Initialize session manager with the chosen adapter
	sessionManager := session.NewManager(storageAdapter)

	telegramBot, err := telegram.NewBot(cfg.TelegramBotToken, sessionManager)
	if err != nil {
		log.Fatalf("Failed to create Telegram bot: %v", err)
	}

	go telegramBot.Start()

	mcpServer := mcp.NewServer()
	hitlHandler := handlers.NewHITLHandler(sessionManager, telegramBot)
	// Pass storageAdapter and cfg to NewRouter
	appRouter := router.NewRouter(mcpServer, hitlHandler, storageAdapter, cfg)

	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      appRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting HTTP server on port %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Close the database connection if a closer function was set
	if closer != nil {
		closer()
	}

	log.Println("Server exited")
}