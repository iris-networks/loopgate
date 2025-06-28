package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type HITLRequest struct {
	ID        string                 `json:"id"`
	ClientID  string                 `json:"client_id"`
	SessionID string                 `json:"session_id"`
	Message   string                 `json:"message"`
	Options   []string               `json:"options,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type HITLResponse struct {
	ID       string    `json:"id"`
	Response string    `json:"response"`
	Selected int       `json:"selected,omitempty"`
	Approved bool      `json:"approved"`
	Time     time.Time `json:"time"`
}

func main() {
	serverURL := "http://localhost:8080"

	sessionID := "test-session-123"
	clientID := "test-client"
	telegramID := int64(12345678)

	fmt.Println("1. Registering session...")
	if err := registerSession(serverURL, sessionID, clientID, telegramID); err != nil {
		log.Fatalf("Failed to register session: %v", err)
	}
	fmt.Println("✅ Session registered successfully")

	fmt.Println("\n2. Sending HITL request...")
	request := HITLRequest{
		ID:        "request-001",
		ClientID:  clientID,
		SessionID: sessionID,
		Message:   "Should I proceed with the deployment to production?",
		Options:   []string{"Deploy", "Cancel", "Review First"},
		Metadata: map[string]interface{}{
			"environment": "production",
			"service":     "api-gateway",
			"version":     "v1.2.3",
		},
	}

	response, err := sendHITLRequest(serverURL, request)
	if err != nil {
		log.Fatalf("Failed to send HITL request: %v", err)
	}

	fmt.Printf("✅ Received response: %s (Approved: %t)\n", response.Response, response.Approved)
}

func registerSession(serverURL, sessionID, clientID string, telegramID int64) error {
	payload := map[string]interface{}{
		"session_id":  sessionID,
		"client_id":   clientID,
		"telegram_id": telegramID,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(serverURL+"/hitl/register", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status: %d", resp.StatusCode)
	}

	return nil
}

func sendHITLRequest(serverURL string, request HITLRequest) (*HITLResponse, error) {
	jsonPayload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(serverURL+"/hitl/request", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	var response HITLResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}