package main

import (
	"context"
	"log"
	"loopgate/config"
	"loopgate/internal/handlers"
	"loopgate/internal/mcp"
	"loopgate/internal/router"
	"loopgate/internal/session"
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

	sessionManager := session.NewManager()

	telegramBot, err := telegram.NewBot(cfg.TelegramBotToken, sessionManager)
	if err != nil {
		log.Fatalf("Failed to create Telegram bot: %v", err)
	}

	go telegramBot.Start()

	mcpServer := mcp.NewServer()
	hitlHandler := handlers.NewHITLHandler(sessionManager, telegramBot)
	appRouter := router.NewRouter(mcpServer, hitlHandler)

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

	log.Println("Server exited")
}