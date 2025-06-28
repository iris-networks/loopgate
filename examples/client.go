package main

import (
	"fmt"
	"log"
	"loopgate/pkg/client"
)

func main() {
	mcpClient := client.NewMCPClient()

	err := mcpClient.ConnectToServer("./loopgate")
	if err != nil {
		log.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer mcpClient.Close()

	err = mcpClient.Initialize("ExampleClient", "1.0.0")
	if err != nil {
		log.Fatalf("Failed to initialize MCP client: %v", err)
	}

	fmt.Println("Connected to Loopgate MCP server!")

	tools, err := mcpClient.ListTools()
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("Available tools: %d\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}

	fmt.Println("\nSending HITL request...")
	
	response, err := mcpClient.SendHITLRequest(
		"example-session",
		"example-client",
		"Should we proceed with the deployment?",
		[]string{"Deploy", "Cancel", "Review"},
		map[string]interface{}{
			"version":     "v1.2.3",
			"environment": "production",
		},
	)
	
	if err != nil {
		log.Fatalf("Failed to send HITL request: %v", err)
	}

	fmt.Printf("HITL response: %+v\n", response)
}

func exampleHTTPClient() {
	httpExample := `
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HITLRequest struct {
	SessionID string                 ` + "`json:\"session_id\"`" + `
	ClientID  string                 ` + "`json:\"client_id\"`" + `
	Message   string                 ` + "`json:\"message\"`" + `
	Options   []string               ` + "`json:\"options,omitempty\"`" + `
	Metadata  map[string]interface{} ` + "`json:\"metadata,omitempty\"`" + `
}

type SessionRegistration struct {
	SessionID  string ` + "`json:\"session_id\"`" + `
	ClientID   string ` + "`json:\"client_id\"`" + `
	TelegramID int64  ` + "`json:\"telegram_id\"`" + `
}

func main() {
	baseURL := "http://localhost:8080"
	
	// Register session
	regReq := SessionRegistration{
		SessionID:  "production-deploy-bot",
		ClientID:   "ci-cd-pipeline",
		TelegramID: 123456789, // Your Telegram user ID
	}
	
	regJSON, _ := json.Marshal(regReq)
	resp, err := http.Post(baseURL+"/hitl/register", "application/json", bytes.NewBuffer(regJSON))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	
	fmt.Println("Session registered")
	
	// Send HITL request
	hitlReq := HITLRequest{
		SessionID: "production-deploy-bot",
		ClientID:  "ci-cd-pipeline",
		Message:   "Deploy v2.1.0 to production? All tests passed ✅",
		Options:   []string{"🚀 Deploy", "⏸️ Hold", "🔍 Review First"},
		Metadata: map[string]interface{}{
			"version":      "v2.1.0",
			"tests_passed": 847,
			"environment":  "production",
		},
	}
	
	hitlJSON, _ := json.Marshal(hitlReq)
	resp, err = http.Post(baseURL+"/hitl/request", "application/json", bytes.NewBuffer(hitlJSON))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	requestID := result["request_id"].(string)
	
	fmt.Printf("Request submitted: %s\n", requestID)
	
	// Poll for response
	for {
		pollResp, err := http.Get(fmt.Sprintf("%s/hitl/poll?request_id=%s", baseURL, requestID))
		if err != nil {
			panic(err)
		}
		
		body, _ := io.ReadAll(pollResp.Body)
		pollResp.Body.Close()
		
		var status map[string]interface{}
		json.Unmarshal(body, &status)
		
		if status["completed"].(bool) {
			if status["approved"].(bool) {
				fmt.Printf("✅ Approved: %s\n", status["response"])
			} else {
				fmt.Printf("❌ Denied: %s\n", status["response"])
			}
			break
		}
		
		fmt.Println("⏳ Waiting for human response...")
		time.Sleep(5 * time.Second)
	}
}
`
	fmt.Println("HTTP client example:")
	fmt.Println(httpExample)
}