package mcp

import (
	"encoding/json"
	"fmt"
)

type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type MCPNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

const (
	MCPVersion = "2.0"
	
	MethodInitialize        = "initialize"
	MethodInitialized       = "initialized"
	MethodShutdown         = "shutdown"
	MethodExit             = "exit"
	MethodListTools        = "tools/list"
	MethodCallTool         = "tools/call"
	MethodListResources    = "resources/list"
	MethodReadResource     = "resources/read"
	MethodSubscribe        = "resources/subscribe"
	MethodUnsubscribe      = "resources/unsubscribe"
	MethodListPrompts      = "prompts/list"
	MethodGetPrompt        = "prompts/get"
	MethodSetLevel         = "logging/setLevel"
	
	MethodHITLRequest      = "hitl/request"
	MethodHITLRegister     = "hitl/register_session"
	MethodHITLStatus       = "hitl/status"
	MethodHITLDeactivate   = "hitl/deactivate"
)

const (
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603
	ErrorCodeServerError    = -32000
)

type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    ClientCapabilities     `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
	Meta           map[string]interface{} `json:"_meta,omitempty"`
}

type ClientCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Sampling     interface{}            `json:"sampling,omitempty"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Logging   interface{}          `json:"logging,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []Content               `json:"content,omitempty"`
	IsError bool                   `json:"isError,omitempty"`
	Meta    map[string]interface{} `json:"_meta,omitempty"`
}

type Content struct {
	Type string      `json:"type"`
	Text string      `json:"text,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

func NewMCPRequest(method string, params interface{}) *MCPRequest {
	return &MCPRequest{
		JSONRPC: MCPVersion,
		Method:  method,
		Params:  params,
	}
}

func NewMCPResponse(id interface{}, result interface{}) *MCPResponse {
	return &MCPResponse{
		JSONRPC: MCPVersion,
		ID:      id,
		Result:  result,
	}
}

func NewMCPError(id interface{}, code int, message string, data interface{}) *MCPResponse {
	return &MCPResponse{
		JSONRPC: MCPVersion,
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func NewMCPNotification(method string, params interface{}) *MCPNotification {
	return &MCPNotification{
		JSONRPC: MCPVersion,
		Method:  method,
		Params:  params,
	}
}

func ParseMCPMessage(data []byte) (*MCPRequest, error) {
	var req MCPRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to parse MCP message: %v", err)
	}
	
	if req.JSONRPC != MCPVersion {
		return nil, fmt.Errorf("unsupported JSON-RPC version: %s", req.JSONRPC)
	}
	
	return &req, nil
}