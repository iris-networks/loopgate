package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"loopgate/internal/types"
)

type MCPServer struct {
	router        HITLRouter
	initialized   bool
	mu            sync.RWMutex
	capabilities  ServerCapabilities
	serverInfo    ServerInfo
	tools         []Tool
}

type HITLRouter interface {
	RouteHITLRequest(req *types.HITLRequest) (*types.HITLResponse, error)
	HandleTelegramResponse(sessionID string, response *types.HITLResponse) error
}

func NewMCPServer(router HITLRouter) *MCPServer {
	server := &MCPServer{
		router: router,
		serverInfo: ServerInfo{
			Name:    "loopgate",
			Version: "1.0.0",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
		},
	}

	server.tools = []Tool{
		{
			Name:        "hitl_request",
			Description: "Send a human-in-the-loop request for approval or input",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Unique session identifier",
					},
					"client_id": map[string]interface{}{
						"type":        "string",
						"description": "Client identifier",
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Message to send to human operator",
					},
					"options": map[string]interface{}{
						"type":        "array",
						"description": "Optional list of choices for user",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "Additional metadata for the request",
					},
				},
				"required": []string{"session_id", "client_id", "message"},
			},
		},
		{
			Name:        "register_session",
			Description: "Register a new HITL session",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Unique session identifier",
					},
					"client_id": map[string]interface{}{
						"type":        "string",
						"description": "Client identifier",
					},
					"telegram_id": map[string]interface{}{
						"type":        "integer",
						"description": "Telegram chat ID",
					},
				},
				"required": []string{"session_id", "client_id", "telegram_id"},
			},
		},
	}

	return server
}

func (s *MCPServer) HandleStdio(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	scanner := bufio.NewScanner(stdin)
	encoder := json.NewEncoder(stdout)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		request, err := ParseMCPMessage(line)
		if err != nil {
			response := NewMCPError(nil, ErrorCodeParseError, err.Error(), nil)
			encoder.Encode(response)
			continue
		}

		response := s.handleMCPRequest(request)
		if err := encoder.Encode(response); err != nil {
			log.Printf("Failed to encode response: %v", err)
		}
	}

	return scanner.Err()
}

func (s *MCPServer) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := NewMCPError(nil, ErrorCodeParseError, err.Error(), nil)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := s.handleMCPRequest(&request)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *MCPServer) handleMCPRequest(request *MCPRequest) interface{} {
	switch request.Method {
	case MethodInitialize:
		return s.handleInitialize(request)
	case MethodInitialized:
		return s.handleInitialized(request)
	case MethodListTools:
		return s.handleListTools(request)
	case MethodCallTool:
		return s.handleCallTool(request)
	case MethodShutdown:
		return s.handleShutdown(request)
	default:
		return NewMCPError(request.ID, ErrorCodeMethodNotFound, 
			fmt.Sprintf("Method not found: %s", request.Method), nil)
	}
}

func (s *MCPServer) handleInitialize(request *MCPRequest) interface{} {
	s.mu.Lock()
	s.initialized = true
	s.mu.Unlock()

	result := InitializeResult{
		ProtocolVersion: MCPVersion,
		Capabilities:    s.capabilities,
		ServerInfo:      s.serverInfo,
		Instructions:    "Loopgate MCP Server for Human-in-the-Loop workflows. Use hitl_request tool to send requests to human operators via Telegram.",
	}

	return NewMCPResponse(request.ID, result)
}

func (s *MCPServer) handleInitialized(request *MCPRequest) interface{} {
	return nil
}

func (s *MCPServer) handleListTools(request *MCPRequest) interface{} {
	s.mu.RLock()
	if !s.initialized {
		s.mu.RUnlock()
		return NewMCPError(request.ID, ErrorCodeInvalidRequest, "Server not initialized", nil)
	}
	s.mu.RUnlock()

	result := map[string]interface{}{
		"tools": s.tools,
	}

	return NewMCPResponse(request.ID, result)
}

func (s *MCPServer) handleCallTool(request *MCPRequest) interface{} {
	s.mu.RLock()
	if !s.initialized {
		s.mu.RUnlock()
		return NewMCPError(request.ID, ErrorCodeInvalidRequest, "Server not initialized", nil)
	}
	s.mu.RUnlock()

	params, ok := request.Params.(map[string]interface{})
	if !ok {
		return NewMCPError(request.ID, ErrorCodeInvalidParams, "Invalid parameters", nil)
	}

	toolName, ok := params["name"].(string)
	if !ok {
		return NewMCPError(request.ID, ErrorCodeInvalidParams, "Missing tool name", nil)
	}

	arguments, _ := params["arguments"].(map[string]interface{})

	switch toolName {
	case "hitl_request":
		return s.handleHITLToolCall(request.ID, arguments)
	case "register_session":
		return s.handleRegisterSessionToolCall(request.ID, arguments)
	default:
		return NewMCPError(request.ID, ErrorCodeMethodNotFound, 
			fmt.Sprintf("Tool not found: %s", toolName), nil)
	}
}

func (s *MCPServer) handleHITLToolCall(requestID interface{}, args map[string]interface{}) interface{} {
	sessionID := getString(args, "session_id")
	clientID := getString(args, "client_id")
	message := getString(args, "message")

	if sessionID == "" || clientID == "" || message == "" {
		return NewMCPError(requestID, ErrorCodeInvalidParams, 
			"Missing required parameters: session_id, client_id, message", nil)
	}

	req := &types.HITLRequest{
		ID:        fmt.Sprintf("hitl_%d", time.Now().UnixNano()),
		ClientID:  clientID,
		SessionID: sessionID,
		Message:   message,
		Options:   getStringSlice(args, "options"),
		Metadata:  getMap(args, "metadata"),
		Timestamp: time.Now(),
	}

	response, err := s.router.RouteHITLRequest(req)
	if err != nil {
		return NewMCPError(requestID, ErrorCodeServerError, err.Error(), nil)
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("HITL Response: %s (Approved: %t)", response.Response, response.Approved),
		},
	}

	result := CallToolResult{
		Content: content,
		IsError: false,
		Meta: map[string]interface{}{
			"response_id": response.ID,
			"timestamp":  response.Time,
			"approved":   response.Approved,
		},
	}

	return NewMCPResponse(requestID, result)
}

func (s *MCPServer) handleRegisterSessionToolCall(requestID interface{}, args map[string]interface{}) interface{} {
	return NewMCPError(requestID, ErrorCodeMethodNotFound, 
		"Use /hitl/register HTTP endpoint for session registration", nil)
}

func (s *MCPServer) handleShutdown(request *MCPRequest) interface{} {
	return NewMCPResponse(request.ID, nil)
}

func getString(params map[string]interface{}, key string) string {
	if val, ok := params[key].(string); ok {
		return val
	}
	return ""
}

func getStringSlice(params map[string]interface{}, key string) []string {
	if val, ok := params[key].([]interface{}); ok {
		result := make([]string, len(val))
		for i, v := range val {
			if str, ok := v.(string); ok {
				result[i] = str
			}
		}
		return result
	}
	return nil
}

func getMap(params map[string]interface{}, key string) map[string]interface{} {
	if val, ok := params[key].(map[string]interface{}); ok {
		return val
	}
	return nil
}