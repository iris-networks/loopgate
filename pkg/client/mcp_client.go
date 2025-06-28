package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"loopgate/internal/mcp"
)

type MCPClient struct {
	serverCmd    *exec.Cmd
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	encoder      *json.Encoder
	decoder      *json.Decoder
	requestID    int64
	mu           sync.Mutex
	initialized  bool
	capabilities *mcp.ServerCapabilities
	tools        []mcp.Tool
}

type HITLRequest struct {
	SessionID string                 `json:"session_id"`
	ClientID  string                 `json:"client_id"`
	Message   string                 `json:"message"`
	Options   []string               `json:"options,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type HITLResponse struct {
	Response string    `json:"response"`
	Approved bool      `json:"approved"`
	Selected int       `json:"selected,omitempty"`
	Time     time.Time `json:"time"`
}

func NewMCPClient() *MCPClient {
	return &MCPClient{
		requestID: 1,
	}
}

func (c *MCPClient) ConnectToServer(serverPath string, args ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cmd := exec.Command(serverPath, args...)
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %v", err)
	}
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %v", err)
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	c.serverCmd = cmd
	c.stdin = stdin
	c.stdout = stdout
	c.stderr = stderr
	c.encoder = json.NewEncoder(stdin)
	c.decoder = json.NewDecoder(stdout)

	return nil
}

func (c *MCPClient) ConnectHTTP(baseURL string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	return fmt.Errorf("HTTP MCP client not yet implemented")
}

func (c *MCPClient) Initialize(clientName, clientVersion string) error {
	params := mcp.InitializeParams{
		ProtocolVersion: mcp.MCPVersion,
		Capabilities: mcp.ClientCapabilities{
			Experimental: make(map[string]interface{}),
		},
		ClientInfo: mcp.ClientInfo{
			Name:    clientName,
			Version: clientVersion,
		},
	}

	response, err := c.sendRequest(mcp.MethodInitialize, params)
	if err != nil {
		return err
	}

	if response.Error != nil {
		return fmt.Errorf("initialization failed: %s", response.Error.Message)
	}

	var result mcp.InitializeResult
	if err := json.Unmarshal(response.Result.(json.RawMessage), &result); err != nil {
		return fmt.Errorf("failed to parse initialize result: %v", err)
	}

	c.capabilities = &result.Capabilities
	c.initialized = true

	if err := c.sendNotification(mcp.MethodInitialized, nil); err != nil {
		return fmt.Errorf("failed to send initialized notification: %v", err)
	}

	if err := c.loadTools(); err != nil {
		return fmt.Errorf("failed to load tools: %v", err)
	}

	return nil
}

func (c *MCPClient) loadTools() error {
	response, err := c.sendRequest(mcp.MethodListTools, nil)
	if err != nil {
		return err
	}

	if response.Error != nil {
		return fmt.Errorf("list tools failed: %s", response.Error.Message)
	}

	var result struct {
		Tools []mcp.Tool `json:"tools"`
	}
	if err := json.Unmarshal(response.Result.(json.RawMessage), &result); err != nil {
		return fmt.Errorf("failed to parse tools result: %v", err)
	}

	c.tools = result.Tools
	return nil
}

func (c *MCPClient) SendHITLRequest(ctx context.Context, req HITLRequest) (*HITLResponse, error) {
	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := mcp.CallToolParams{
		Name: "hitl_request",
		Arguments: map[string]interface{}{
			"session_id": req.SessionID,
			"client_id":  req.ClientID,
			"message":    req.Message,
			"options":    req.Options,
			"metadata":   req.Metadata,
		},
	}

	response, err := c.sendRequest(mcp.MethodCallTool, params)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, fmt.Errorf("HITL request failed: %s", response.Error.Message)
	}

	var result mcp.CallToolResult
	if err := json.Unmarshal(response.Result.(json.RawMessage), &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %v", err)
	}

	if result.IsError {
		return nil, fmt.Errorf("tool call failed")
	}

	approved, _ := result.Meta["approved"].(bool)
	timestamp, _ := result.Meta["timestamp"].(string)
	
	var responseTime time.Time
	if timestamp != "" {
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			responseTime = t
		}
	}

	responseText := ""
	if len(result.Content) > 0 {
		responseText = result.Content[0].Text
	}

	return &HITLResponse{
		Response: responseText,
		Approved: approved,
		Time:     responseTime,
	}, nil
}

func (c *MCPClient) GetAvailableTools() []mcp.Tool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tools
}

func (c *MCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.serverCmd != nil {
		c.sendRequest(mcp.MethodShutdown, nil)
		c.stdin.Close()
		c.stdout.Close()
		c.stderr.Close()
		return c.serverCmd.Wait()
	}

	return nil
}

func (c *MCPClient) sendRequest(method string, params interface{}) (*mcp.MCPResponse, error) {
	c.mu.Lock()
	requestID := c.requestID
	c.requestID++
	c.mu.Unlock()

	request := mcp.NewMCPRequest(method, params)
	request.ID = requestID

	if err := c.encoder.Encode(request); err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	var response mcp.MCPResponse
	if err := c.decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	return &response, nil
}

func (c *MCPClient) sendNotification(method string, params interface{}) error {
	notification := mcp.NewMCPNotification(method, params)
	return c.encoder.Encode(notification)
}

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (hc *HTTPClient) RegisterSession(sessionID, clientID string, telegramID int64) error {
	payload := map[string]interface{}{
		"session_id":  sessionID,
		"client_id":   clientID,
		"telegram_id": telegramID,
	}

	_, err := hc.post("/hitl/register", payload)
	return err
}

func (hc *HTTPClient) SendHITLRequest(req HITLRequest) (*HITLResponse, error) {
	response, err := hc.post("/hitl/request", req)
	if err != nil {
		return nil, err
	}

	var hitlResp HITLResponse
	if err := json.Unmarshal(response, &hitlResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &hitlResp, nil
}

func (hc *HTTPClient) GetSessionStatus(sessionID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/hitl/status?session_id=%s", hc.baseURL, sessionID)
	
	resp, err := hc.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (hc *HTTPClient) post(endpoint string, payload interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := hc.baseURL + endpoint
	resp, err := hc.httpClient.Post(url, "application/json", 
		bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}