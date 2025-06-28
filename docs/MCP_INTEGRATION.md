# MCP Integration Guide

This guide explains how to integrate Loopgate with various AI systems using the Model Context Protocol (MCP).

## Overview

Loopgate implements MCP 2.0 specification and exposes the following tools:

- `request_human_input` - Request human approval/input
- `check_request_status` - Poll request status
- `list_pending_requests` - List pending requests
- `cancel_request` - Cancel a request

## Basic MCP Usage

### 1. Start MCP Server

```bash
# Start as standalone MCP server (stdio)
./loopgate

# Or start as HTTP server with MCP endpoint
make run
```

### 2. Connect MCP Client

```go
import "loopgate/pkg/client"

client := client.NewMCPClient()
err := client.ConnectToServer("./loopgate")
if err != nil {
    log.Fatal(err)
}

err = client.Initialize("MyAI", "1.0.0")
if err != nil {
    log.Fatal(err)
}
```

### 3. List Available Tools

```go
tools, err := client.ListTools()
if err != nil {
    log.Fatal(err)
}

for _, tool := range tools {
    fmt.Printf("Tool: %s - %s\n", tool.Name, tool.Description)
}
```

### 4. Request Human Input

```go
response, err := client.SendHITLRequest(
    "my-session",
    "my-ai-agent", 
    "Should I proceed with deployment?",
    []string{"Yes", "No", "Review First"},
    map[string]interface{}{
        "environment": "production",
        "version": "v1.2.3",
    },
)
```

## Integration with AI Frameworks

### Claude Desktop

Add to your Claude Desktop MCP configuration:

```json
{
  "mcpServers": {
    "loopgate": {
      "command": "/path/to/loopgate",
      "args": []
    }
  }
}
```

### Custom Integration

```typescript
import { MCPClient } from '@modelcontextprotocol/sdk/client';

const client = new MCPClient({
  name: "my-ai-agent",
  version: "1.0.0"
});

await client.connect({
  command: "./loopgate",
  args: []
});

// Request human approval
const result = await client.callTool("request_human_input", {
  session_id: "my-session",
  client_id: "my-ai",
  message: "Approve this action?",
  options: ["Approve", "Deny"]
});
```

## HTTP MCP Endpoint

For systems that prefer HTTP:

```bash
# POST to /mcp endpoint
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "request_human_input",
      "arguments": {
        "session_id": "test",
        "client_id": "test",
        "message": "Test message"
      }
    },
    "id": "1"
  }'
```

## Error Handling

MCP responses include error information:

```json
{
  "error": {
    "code": -32601,
    "message": "Method not found",
    "data": null
  },
  "id": "1"
}
```

Common error codes:
- `-32700`: Parse error
- `-32600`: Invalid request  
- `-32601`: Method not found
- `-32602`: Invalid params

## Best Practices

1. **Session Management**: Always register sessions before sending requests
2. **Error Handling**: Check for MCP errors in responses
3. **Timeouts**: Set appropriate timeouts for requests
4. **Cleanup**: Close MCP connections properly
5. **Logging**: Log MCP interactions for debugging

## Troubleshooting

### Connection Issues
- Ensure Loopgate binary is executable
- Check that stdio pipes are working
- Verify process permissions

### Tool Call Failures  
- Validate tool parameters against schema
- Check session registration status
- Verify Telegram bot configuration

### Performance
- Use HTTP mode for high-throughput scenarios
- Implement connection pooling for multiple agents
- Monitor memory usage with many concurrent requests