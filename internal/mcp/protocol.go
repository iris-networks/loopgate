package mcp

import (
	"encoding/json"
	"loopgate/internal/types"
)

const (
	ProtocolVersion = "2024-11-05"
	ServerName      = "loopgate"
	ServerVersion   = "1.0.0"
)

type Protocol struct{}

func NewProtocol() *Protocol {
	return &Protocol{}
}

func (p *Protocol) HandleRequest(requestData []byte) ([]byte, error) {
	var req types.MCPRequest
	if err := json.Unmarshal(requestData, &req); err != nil {
		return p.createErrorResponse(nil, -32700, "Parse error", nil)
	}

	switch req.Method {
	case "initialize":
		return p.handleInitialize(req)
	case "tools/list":
		return p.handleToolsList(req)
	case "tools/call":
		return p.handleToolsCall(req)
	default:
		return p.createErrorResponse(req.ID, -32601, "Method not found", nil)
	}
}

func (p *Protocol) handleInitialize(req types.MCPRequest) ([]byte, error) {
	result := types.MCPInitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: types.MCPCapabilities{
			Tools: map[string]interface{}{
				"listChanged": true,
			},
		},
		ServerInfo: types.MCPServerInfo{
			Name:    ServerName,
			Version: ServerVersion,
		},
	}

	return p.createSuccessResponse(req.ID, result)
}

func (p *Protocol) handleToolsList(req types.MCPRequest) ([]byte, error) {
	tools := []types.MCPTool{
		{
			Name:        "request_human_input",
			Description: "Request human input for decision making or approval",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"client_id": map[string]interface{}{
						"type":        "string",
						"description": "Unique identifier for the AI client",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session identifier for routing",
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Message to display to the human",
					},
					"request_type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"confirmation", "input", "choice"},
						"description": "Type of human input requested",
					},
					"options": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Available choices for choice type requests",
					},
					"timeout_seconds": map[string]interface{}{
						"type":        "number",
						"description": "Request timeout in seconds",
						"default":     300,
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "Additional metadata for the request",
					},
				},
				"required": []string{"client_id", "session_id", "message"},
			},
		},
		{
			Name:        "check_request_status",
			Description: "Check the status of a human input request",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"request_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the request to check",
					},
				},
				"required": []string{"request_id"},
			},
		},
		{
			Name:        "list_pending_requests",
			Description: "List all pending human input requests",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "cancel_request",
			Description: "Cancel a pending human input request",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"request_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the request to cancel",
					},
				},
				"required": []string{"request_id"},
			},
		},
	}

	result := map[string]interface{}{
		"tools": tools,
	}

	return p.createSuccessResponse(req.ID, result)
}

func (p *Protocol) handleToolsCall(req types.MCPRequest) ([]byte, error) {
	paramsMap, ok := req.Params.(map[string]interface{})
	if !ok {
		return p.createErrorResponse(req.ID, -32602, "Invalid params", nil)
	}

	toolName, ok := paramsMap["name"].(string)
	if !ok {
		return p.createErrorResponse(req.ID, -32602, "Missing tool name", nil)
	}

	arguments, ok := paramsMap["arguments"].(map[string]interface{})
	if !ok {
		arguments = make(map[string]interface{})
	}

	switch toolName {
	case "request_human_input":
		return p.handleRequestHumanInput(req.ID, arguments)
	case "check_request_status":
		return p.handleCheckRequestStatus(req.ID, arguments)
	case "list_pending_requests":
		return p.handleListPendingRequests(req.ID, arguments)
	case "cancel_request":
		return p.handleCancelRequest(req.ID, arguments)
	default:
		return p.createErrorResponse(req.ID, -32601, "Tool not found", nil)
	}
}

func (p *Protocol) handleRequestHumanInput(requestID interface{}, args map[string]interface{}) ([]byte, error) {
	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": "MCP tool call received. Use HTTP API endpoint /hitl/request to submit the actual request.",
			},
		},
		"isError": false,
	}

	return p.createSuccessResponse(requestID, result)
}

func (p *Protocol) handleCheckRequestStatus(requestID interface{}, args map[string]interface{}) ([]byte, error) {
	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": "Use HTTP API endpoint /hitl/poll?request_id=<id> to check request status.",
			},
		},
		"isError": false,
	}

	return p.createSuccessResponse(requestID, result)
}

func (p *Protocol) handleListPendingRequests(requestID interface{}, args map[string]interface{}) ([]byte, error) {
	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": "Use HTTP API endpoint /hitl/pending to list pending requests.",
			},
		},
		"isError": false,
	}

	return p.createSuccessResponse(requestID, result)
}

func (p *Protocol) handleCancelRequest(requestID interface{}, args map[string]interface{}) ([]byte, error) {
	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": "Use HTTP API endpoint /hitl/cancel to cancel a request.",
			},
		},
		"isError": false,
	}

	return p.createSuccessResponse(requestID, result)
}

func (p *Protocol) createSuccessResponse(id interface{}, result interface{}) ([]byte, error) {
	response := types.MCPResponse{
		Result: result,
		ID:     id,
	}

	return json.Marshal(response)
}

func (p *Protocol) createErrorResponse(id interface{}, code int, message string, data interface{}) ([]byte, error) {
	response := types.MCPResponse{
		Error: &types.MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}

	return json.Marshal(response)
}

func (p *Protocol) GetAvailableTools() []types.MCPTool {
	tools := []types.MCPTool{
		{
			Name:        "request_human_input",
			Description: "Request human input for decision making or approval",
		},
		{
			Name:        "check_request_status",
			Description: "Check the status of a human input request",
		},
		{
			Name:        "list_pending_requests",
			Description: "List all pending human input requests",
		},
		{
			Name:        "cancel_request",
			Description: "Cancel a pending human input request",
		},
	}

	return tools
}