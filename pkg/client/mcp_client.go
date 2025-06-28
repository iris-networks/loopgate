package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"loopgate/internal/types"
	"os"
	"os/exec"
)

type MCPClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	reader *bufio.Reader
}

func NewMCPClient() *MCPClient {
	return &MCPClient{}
}

func (c *MCPClient) ConnectToServer(serverPath string) error {
	c.cmd = exec.Command(serverPath)
	
	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	c.reader = bufio.NewReader(c.stdout)

	c.cmd.Stderr = os.Stderr

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	return nil
}

func (c *MCPClient) Initialize(clientName, clientVersion string) error {
	initRequest := types.MCPRequest{
		Method: "initialize",
		Params: types.MCPInitializeParams{
			ProtocolVersion: "2024-11-05",
			Capabilities: types.MCPCapabilities{
				Tools: map[string]interface{}{
					"listChanged": true,
				},
			},
			ClientInfo: types.MCPClientInfo{
				Name:    clientName,
				Version: clientVersion,
			},
		},
		ID: "init-1",
	}

	_, err := c.sendRequest(initRequest)
	return err
}

func (c *MCPClient) ListTools() ([]types.MCPTool, error) {
	request := types.MCPRequest{
		Method: "tools/list",
		Params: map[string]interface{}{},
		ID:     "tools-list-1",
	}

	response, err := c.sendRequest(request)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", response.Error.Message)
	}

	resultMap, ok := response.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	toolsData, ok := resultMap["tools"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid tools format")
	}

	var tools []types.MCPTool
	for _, toolData := range toolsData {
		toolBytes, err := json.Marshal(toolData)
		if err != nil {
			continue
		}

		var tool types.MCPTool
		if err := json.Unmarshal(toolBytes, &tool); err != nil {
			continue
		}

		tools = append(tools, tool)
	}

	return tools, nil
}

func (c *MCPClient) CallTool(name string, arguments map[string]interface{}) (*types.MCPResponse, error) {
	request := types.MCPRequest{
		Method: "tools/call",
		Params: map[string]interface{}{
			"name":      name,
			"arguments": arguments,
		},
		ID: fmt.Sprintf("tool-call-%s", name),
	}

	return c.sendRequest(request)
}

func (c *MCPClient) SendHITLRequest(sessionID, clientID, message string, options []string, metadata map[string]interface{}) (*types.MCPResponse, error) {
	args := map[string]interface{}{
		"session_id": sessionID,
		"client_id":  clientID,
		"message":    message,
	}

	if len(options) > 0 {
		args["options"] = options
		args["request_type"] = "choice"
	} else {
		args["request_type"] = "input"
	}

	if metadata != nil {
		args["metadata"] = metadata
	}

	return c.CallTool("request_human_input", args)
}

func (c *MCPClient) CheckRequestStatus(requestID string) (*types.MCPResponse, error) {
	args := map[string]interface{}{
		"request_id": requestID,
	}

	return c.CallTool("check_request_status", args)
}

func (c *MCPClient) sendRequest(request types.MCPRequest) (*types.MCPResponse, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if _, err := c.stdin.Write(append(requestBytes, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	responseBytes, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response types.MCPResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

func (c *MCPClient) Close() error {
	if c.stdin != nil {
		c.stdin.Close()
	}

	if c.stdout != nil {
		c.stdout.Close()
	}

	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}

	return nil
}