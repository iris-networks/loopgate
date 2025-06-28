# Loopgate Usage Guide

## Overview

Loopgate is a Model Context Protocol (MCP) server that enables AI agents to request human input and approval through Telegram. This guide covers how to integrate your AI agents with Loopgate.

## Quick Start

### 1. Setup Loopgate Server

```bash
# Clone and build
git clone <your-repo-url>
cd loopgate
make build

# Set environment variables
export TELEGRAM_BOT_TOKEN="your_bot_token_here"
export SERVER_PORT=8080

# Run the server
make run
```

### 2. Register a Session

Before your AI agent can send HITL requests, you need to register a session:

```bash
curl -X POST http://localhost:8080/hitl/register \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "my-ai-agent-001",
    "client_id": "production-assistant",
    "telegram_id": 123456789
  }'
```

### 3. Send HITL Requests

#### Using HTTP API

```bash
curl -X POST http://localhost:8080/hitl/request \
  -H "Content-Type: application/json" \
  -d '{
    "id": "req-001",
    "session_id": "my-ai-agent-001",
    "client_id": "production-assistant",
    "message": "Should I deploy the new feature to production?",
    "options": ["Deploy", "Cancel", "Review First"],
    "metadata": {
      "service": "api-gateway",
      "version": "v2.1.0",
      "environment": "production"
    }
  }'
```

#### Using MCP Protocol

```python
import json
import subprocess

# Start Loopgate as MCP server
process = subprocess.Popen(['./build/loopgate'], 
                          stdin=subprocess.PIPE, 
                          stdout=subprocess.PIPE, 
                          text=True)

# Initialize MCP connection
init_request = {
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
        "protocolVersion": "2.0",
        "clientInfo": {"name": "MyAI", "version": "1.0"},
        "capabilities": {}
    }
}

process.stdin.write(json.dumps(init_request) + '\n')
process.stdin.flush()

# Send HITL request using MCP tool
hitl_request = {
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
        "name": "hitl_request",
        "arguments": {
            "session_id": "my-ai-agent-001",
            "client_id": "production-assistant",
            "message": "Should I deploy to production?",
            "options": ["Deploy", "Cancel", "Review"]
        }
    }
}

process.stdin.write(json.dumps(hitl_request) + '\n')
process.stdin.flush()
```

## Integration Examples

### Python AI Agent

```python
import requests
import json
from datetime import datetime

class LoopgateClient:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
        self.session_id = None
        self.client_id = None
    
    def register_session(self, session_id, client_id, telegram_id):
        response = requests.post(f"{self.base_url}/hitl/register", json={
            "session_id": session_id,
            "client_id": client_id,
            "telegram_id": telegram_id
        })
        response.raise_for_status()
        self.session_id = session_id
        self.client_id = client_id
        return response.json()
    
    def request_approval(self, message, options=None, metadata=None):
        payload = {
            "id": f"req_{int(datetime.now().timestamp())}",
            "session_id": self.session_id,
            "client_id": self.client_id,
            "message": message,
        }
        
        if options:
            payload["options"] = options
        if metadata:
            payload["metadata"] = metadata
            
        response = requests.post(f"{self.base_url}/hitl/request", json=payload)
        response.raise_for_status()
        return response.json()

# Usage
client = LoopgateClient()
client.register_session("ai-agent-1", "deployment-bot", 123456789)

# Request approval for deployment
result = client.request_approval(
    "Deploy new ML model to production?",
    options=["Deploy", "Cancel", "Schedule Later"],
    metadata={
        "model": "recommendation-v2.1",
        "accuracy": "94.2%",
        "environment": "production"
    }
)

if result["approved"]:
    print("Deployment approved!")
else:
    print(f"Deployment denied: {result['response']}")
```

### Node.js AI Agent

```javascript
const axios = require('axios');

class LoopgateClient {
    constructor(baseURL = 'http://localhost:8080') {
        this.baseURL = baseURL;
        this.sessionId = null;
        this.clientId = null;
    }
    
    async registerSession(sessionId, clientId, telegramId) {
        const response = await axios.post(`${this.baseURL}/hitl/register`, {
            session_id: sessionId,
            client_id: clientId,
            telegram_id: telegramId
        });
        
        this.sessionId = sessionId;
        this.clientId = clientId;
        return response.data;
    }
    
    async requestApproval(message, options = null, metadata = null) {
        const payload = {
            id: `req_${Date.now()}`,
            session_id: this.sessionId,
            client_id: this.clientId,
            message: message
        };
        
        if (options) payload.options = options;
        if (metadata) payload.metadata = metadata;
        
        const response = await axios.post(`${this.baseURL}/hitl/request`, payload);
        return response.data;
    }
}

// Usage
async function main() {
    const client = new LoopgateClient();
    
    await client.registerSession('ai-agent-1', 'trading-bot', 123456789);
    
    const result = await client.requestApproval(
        'Execute large trade order?',
        ['Execute', 'Cancel', 'Reduce Size'],
        {
            symbol: 'AAPL',
            quantity: 10000,
            price: '$150.25',
            value: '$1,502,500'
        }
    );
    
    console.log('Trade approval:', result.approved ? 'APPROVED' : 'DENIED');
}

main().catch(console.error);
```

### Go AI Agent using MCP Client SDK

```go
package main

import (
    "context"
    "log"
    "time"
    
    "loopgate/pkg/client"
)

func main() {
    // Create MCP client
    mcpClient := client.NewMCPClient()
    
    // Connect to Loopgate server
    err := mcpClient.ConnectToServer("./build/loopgate")
    if err != nil {
        log.Fatal(err)
    }
    defer mcpClient.Close()
    
    // Initialize connection
    err = mcpClient.Initialize("DeploymentAgent", "1.0.0")
    if err != nil {
        log.Fatal(err)
    }
    
    // Send HITL request
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    request := client.HITLRequest{
        SessionID: "deployment-agent-1",
        ClientID:  "ci-cd-pipeline",
        Message:   "Deploy version 2.1.0 to production?",
        Options:   []string{"Deploy", "Cancel", "Deploy to Staging First"},
        Metadata: map[string]interface{}{
            "version":     "2.1.0",
            "environment": "production",
            "tests_passed": true,
            "code_coverage": "95%",
        },
    }
    
    response, err := mcpClient.SendHITLRequest(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    
    if response.Approved {
        log.Println("Deployment approved! Proceeding...")
        // Execute deployment logic
    } else {
        log.Printf("Deployment denied: %s", response.Response)
    }
}
```

## API Endpoints

### POST /hitl/register
Register a new session for HITL requests.

**Request:**
```json
{
  "session_id": "unique-session-id",
  "client_id": "your-ai-agent-name", 
  "telegram_id": 123456789
}
```

**Response:**
```json
{
  "status": "registered",
  "session_id": "unique-session-id",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /hitl/request
Send a HITL request to human operator.

**Request:**
```json
{
  "id": "request-id",
  "session_id": "unique-session-id",
  "client_id": "your-ai-agent-name",
  "message": "Human-readable message",
  "options": ["Option 1", "Option 2", "Option 3"],
  "metadata": {
    "key": "value"
  }
}
```

**Response:**
```json
{
  "id": "request-id",
  "response": "Human response text",
  "approved": true,
  "selected": 0,
  "time": "2024-01-15T10:31:45Z"
}
```

### GET /hitl/status
Check session status and pending requests.

**Query Parameters:**
- `session_id`: Session ID to check

**Response:**
```json
{
  "session": {
    "id": "unique-session-id",
    "client_id": "your-ai-agent-name",
    "telegram_id": 123456789,
    "is_active": true,
    "created_at": "2024-01-15T10:00:00Z",
    "last_activity": "2024-01-15T10:30:00Z"
  },
  "pending_requests": 1,
  "requests": [...]
}
```

### POST /hitl/deactivate
Deactivate a session.

**Request:**
```json
{
  "session_id": "unique-session-id"
}
```

### GET /health
Server health check.

**Response:**
```json
{
  "status": "healthy",
  "total_sessions": 5,
  "active_sessions": 3,
  "pending_requests": 2,
  "timestamp": "2024-01-15T10:35:00Z"
}
```

## MCP Protocol Support

Loopgate implements the Model Context Protocol (MCP) for standardized AI agent integration.

### Available Tools

#### hitl_request
Send a human-in-the-loop request for approval.

**Parameters:**
- `session_id` (string, required): Session identifier
- `client_id` (string, required): Client identifier  
- `message` (string, required): Message for human
- `options` (array, optional): Choice options
- `metadata` (object, optional): Additional data

**Returns:**
Content with human response and approval status.

### MCP Connection Examples

#### Via stdio
```bash
./build/loopgate
```

#### Via HTTP
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":...}'
```

## Environment Variables

- `TELEGRAM_BOT_TOKEN`: Your Telegram bot token (required)
- `SERVER_PORT`: HTTP server port (default: 8080)
- `LOG_LEVEL`: Logging level (default: info)

## Error Handling

### Common Error Codes

- `400 Bad Request`: Invalid JSON or missing required fields
- `404 Not Found`: Session not found
- `409 Conflict`: Session already exists
- `500 Internal Server Error`: Server error

### MCP Error Codes

- `-32700`: Parse error
- `-32600`: Invalid request
- `-32601`: Method not found
- `-32602`: Invalid params
- `-32603`: Internal error
- `-32000`: Server error

## Best Practices

1. **Session Management**: Use descriptive session IDs that identify your AI agent
2. **Timeouts**: Implement timeouts for HITL requests (default: 5 minutes)
3. **Error Handling**: Always handle network errors and timeouts gracefully
4. **Metadata**: Include relevant context in metadata for better human decision-making
5. **Options**: Provide clear, actionable options when possible
6. **Logging**: Log HITL requests and responses for audit trails

## Troubleshooting

### Common Issues

1. **Session not found**: Ensure session is registered before sending requests
2. **Telegram not responding**: Check bot token and Telegram user has started chat with bot
3. **Timeout errors**: Increase timeout or check if human operator is available
4. **MCP connection issues**: Verify server is running and protocol version matches

### Debug Mode

Enable debug logging:
```bash
LOG_LEVEL=debug ./build/loopgate
```

### Health Check

Monitor server health:
```bash
curl http://localhost:8080/health
```