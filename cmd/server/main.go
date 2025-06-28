package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"loopgate/config"
	"loopgate/internal/handlers"
	"loopgate/internal/mcp"
	"loopgate/internal/router"
	"loopgate/internal/session"
	"loopgate/internal/telegram"
)

func main() {
	log.Println("Starting Loopgate MCP Server...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	sessionManager := session.NewManager("./data")
	telegramBot := telegram.NewBot(cfg.TelegramBotToken)
	messageRouter := router.NewRouter(sessionManager, telegramBot)
	hitlHandler := handlers.NewHITLHandler(messageRouter, sessionManager)
	mcpServer := mcp.NewMCPServer(messageRouter)
	
	telegramBot.SetMCPHandler(messageRouter)

	http.HandleFunc("/hitl/request", hitlHandler.HandleRequest)
	http.HandleFunc("/hitl/register", hitlHandler.HandleSessionRegistration)
	http.HandleFunc("/hitl/status", hitlHandler.HandleSessionStatus)
	http.HandleFunc("/hitl/deactivate", hitlHandler.HandleSessionDeactivation)
	http.HandleFunc("/health", hitlHandler.HandleHealth)
	http.HandleFunc("/telegram/webhook", telegramBot.HandleWebhook)
	http.HandleFunc("/mcp", mcpServer.HandleHTTP)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{
				"service": "Loopgate MCP Server",
				"version": "1.0.0",
				"status": "running",
				"endpoints": {
					"hitl_request": "/hitl/request",
					"session_register": "/hitl/register", 
					"session_status": "/hitl/status",
					"session_deactivate": "/hitl/deactivate",
					"health": "/health",
					"telegram_webhook": "/telegram/webhook",
					"mcp": "/mcp"
				}
			}`)
		} else {
			http.NotFound(w, r)
		}
	})

	go func() {
		log.Println("Starting Telegram bot polling...")
		telegramBot.StartPolling()
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down Loopgate MCP Server...")
		os.Exit(0)
	}()

	serverAddr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Printf("Loopgate MCP Server listening on %s", serverAddr)
	log.Printf("Configuration: Log Level=%s", cfg.LogLevel)
	
	if strings.Contains(cfg.TelegramBotToken, "***") {
		log.Printf("Telegram Bot Token: %s", cfg.TelegramBotToken[:10]+"***")
	} else {
		log.Printf("Telegram Bot Token: %s***", cfg.TelegramBotToken[:10])
	}

	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}