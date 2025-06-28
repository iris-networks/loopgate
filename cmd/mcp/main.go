package main

import (
	"log"
	"loopgate/internal/mcp"
)

func main() {
	server := mcp.NewServer()
	
	if err := server.Start(); err != nil {
		log.Fatalf("MCP server failed: %v", err)
	}
}