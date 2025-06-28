package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"loopgate/config"
	"loopgate/internal/mcp"
	"loopgate/internal/router"
	"loopgate/internal/session"
	"loopgate/internal/telegram"
)

func main() {
	log.Println("Starting Loopgate as MCP Server...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	sessionManager := session.NewManager("./data")
	telegramBot := telegram.NewBot(cfg.TelegramBotToken)
	messageRouter := router.NewRouter(sessionManager, telegramBot)
	mcpServer := mcp.NewMCPServer(messageRouter)

	telegramBot.SetMCPHandler(messageRouter)

	go func() {
		log.Println("Starting Telegram bot polling...")
		telegramBot.StartPolling()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down Loopgate MCP Server...")
		cancel()
	}()

	log.Println("Loopgate MCP Server ready for stdio communication")
	log.Printf("Telegram Bot Token: %s***", cfg.TelegramBotToken[:10])

	if err := mcpServer.HandleStdio(ctx, os.Stdin, os.Stdout); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}