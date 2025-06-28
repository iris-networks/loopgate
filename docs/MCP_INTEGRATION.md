# Model Context Protocol (MCP) Integration

Loopgate provides full support for the Model Context Protocol (MCP), enabling AI agents to seamlessly request human approval through a standardized interface.

## Overview

The Model Context Protocol allows AI agents to:
- **Connect** to Loopgate via stdio or HTTP
- **Initialize** a secure communication channel
- **Call tools** to send HITL requests to human operators
- **Receive responses** with approval decisions

## Running Loopgate as MCP Server

### Stdio Mode (Recommended)
```bash
# Build both server versions
make build

# Run as MCP server via stdio
./build/loopgate-mcp

# Or use make command
make mcp
```

### HTTP Mode
```bash
# Run HTTP server with MCP endpoint
./build/loopgate

# MCP endpoint available at:
# POST http://localhost:8080/mcp
```

## MCP Client Integration

### 1. Initialize Connection

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2.0",
    "clientInfo": {
      "name": "MyAI",
      "version": "1.0.0"
    },
    "capabilities": {}
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2.0",
    "serverInfo": {
      "name": "loopgate",
      "version": "1.0.0"
    },
    "capabilities": {
      "tools": {
        "listChanged": false
      }
    },
    "instructions": "Loopgate MCP Server for Human-in-the-Loop workflows. Use hitl_request tool to send requests to human operators via Telegram."
  }
}
```

### 2. Send Initialized Notification

```json
{
  "jsonrpc": "2.0",
  "method": "initialized",
  "params": {}
}
```

### 3. List Available Tools

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "hitl_request",
        "description": "Send a human-in-the-loop request for approval or input",
        "inputSchema": {
          "type": "object",
          "properties": {
            "session_id": {
              "type": "string",
              "description": "Unique session identifier"
            },
            "client_id": {
              "type": "string", 
              "description": "Client identifier"
            },
            "message": {
              "type": "string",
              "description": "Message to send to human operator"
            },
            "options": {
              "type": "array",
              "description": "Optional list of choices for user",
              "items": {"type": "string"}
            },
            "metadata": {
              "type": "object",
              "description": "Additional metadata for the request"
            }
          },
          "required": ["session_id", "client_id", "message"]
        }
      }
    ]
  }
}
```

### 4. Call HITL Request Tool

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "hitl_request",
    "arguments": {
      "session_id": "production-deploy-bot",
      "client_id": "ci-cd-pipeline",
      "message": "Deploy v2.1.0 to production? All tests passed ✅",
      "options": ["🚀 Deploy", "⏸️ Hold", "🔍 Review First"],
      "metadata": {
        "version": "v2.1.0",
        "tests_passed": 847,
        "code_coverage": "94.2%",
        "environment": "production"
      }
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "HITL Response: 🚀 Deploy (Approved: true)"
      }
    ],
    "isError": false,
    "_meta": {
      "response_id": "resp_1641234567890",
      "timestamp": "2024-01-15T10:35:00Z",
      "approved": true
    }
  }
}
```

## Language-Specific SDKs

### Python MCP Client

```python
import asyncio
import json
import subprocess
from typing import Dict, List, Optional

class LoopgateMCPClient:
    def __init__(self, server_path: str = "./build/loopgate-mcp"):
        self.server_path = server_path
        self.process = None
        self.request_id = 1
        
    async def connect(self):
        self.process = await asyncio.create_subprocess_exec(
            self.server_path,
            stdin=asyncio.subprocess.PIPE,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        
    async def initialize(self, client_name: str, client_version: str):
        request = {
            "jsonrpc": "2.0",
            "id": self._next_id(),
            "method": "initialize",
            "params": {
                "protocolVersion": "2.0",
                "clientInfo": {"name": client_name, "version": client_version},
                "capabilities": {}
            }
        }
        
        response = await self._send_request(request)
        
        # Send initialized notification
        notification = {
            "jsonrpc": "2.0",
            "method": "initialized",
            "params": {}
        }
        await self._send_notification(notification)
        
        return response
        
    async def request_approval(
        self, 
        session_id: str,
        client_id: str,
        message: str,
        options: Optional[List[str]] = None,
        metadata: Optional[Dict] = None
    ):
        arguments = {
            "session_id": session_id,
            "client_id": client_id,
            "message": message
        }
        
        if options:
            arguments["options"] = options
        if metadata:
            arguments["metadata"] = metadata
            
        request = {
            "jsonrpc": "2.0",
            "id": self._next_id(),
            "method": "tools/call",
            "params": {
                "name": "hitl_request",
                "arguments": arguments
            }
        }
        
        response = await self._send_request(request)
        
        if "error" in response:
            raise Exception(f"HITL request failed: {response['error']['message']}")
            
        return response["result"]
        
    async def _send_request(self, request: dict):
        await self._write_message(request)
        return await self._read_message()
        
    async def _send_notification(self, notification: dict):
        await self._write_message(notification)
        
    async def _write_message(self, message: dict):
        data = json.dumps(message) + '\n'
        self.process.stdin.write(data.encode())
        await self.process.stdin.drain()
        
    async def _read_message(self):
        line = await self.process.stdout.readline()
        return json.loads(line.decode().strip())
        
    def _next_id(self):
        self.request_id += 1
        return self.request_id
        
    async def close(self):
        if self.process:
            self.process.stdin.close()
            await self.process.wait()

# Usage Example
async def main():
    client = LoopgateMCPClient()
    await client.connect()
    
    await client.initialize("DeploymentBot", "1.0.0")
    
    result = await client.request_approval(
        session_id="production-deploy-bot",
        client_id="ci-cd-pipeline",
        message="Deploy v2.1.0 to production?",
        options=["Deploy", "Cancel", "Review"],
        metadata={"version": "v2.1.0", "environment": "production"}
    )
    
    print(f"Approval result: {result}")
    await client.close()

if __name__ == "__main__":
    asyncio.run(main())
```

### Node.js MCP Client

```javascript
const { spawn } = require('child_process');
const { EventEmitter } = require('events');

class LoopgateMCPClient extends EventEmitter {
    constructor(serverPath = './build/loopgate-mcp') {
        super();
        this.serverPath = serverPath;
        this.process = null;
        this.requestId = 1;
        this.pendingRequests = new Map();
    }
    
    async connect() {
        this.process = spawn(this.serverPath, [], {
            stdio: ['pipe', 'pipe', 'pipe']
        });
        
        this.process.stdout.on('data', (data) => {
            const lines = data.toString().split('\n').filter(line => line.trim());
            for (const line of lines) {
                try {
                    const message = JSON.parse(line);
                    this._handleMessage(message);
                } catch (err) {
                    console.error('Failed to parse message:', err);
                }
            }
        });
        
        this.process.stderr.on('data', (data) => {
            console.error('Server stderr:', data.toString());
        });
    }
    
    async initialize(clientName, clientVersion) {
        const request = {
            jsonrpc: '2.0',
            id: this._nextId(),
            method: 'initialize',
            params: {
                protocolVersion: '2.0',
                clientInfo: { name: clientName, version: clientVersion },
                capabilities: {}
            }
        };
        
        const response = await this._sendRequest(request);
        
        // Send initialized notification
        await this._sendNotification({
            jsonrpc: '2.0',
            method: 'initialized',
            params: {}
        });
        
        return response;
    }
    
    async requestApproval(sessionId, clientId, message, options = null, metadata = null) {
        const arguments = { session_id: sessionId, client_id: clientId, message };
        
        if (options) arguments.options = options;
        if (metadata) arguments.metadata = metadata;
        
        const request = {
            jsonrpc: '2.0',
            id: this._nextId(),
            method: 'tools/call',
            params: {
                name: 'hitl_request',
                arguments
            }
        };
        
        const response = await this._sendRequest(request);
        
        if (response.error) {
            throw new Error(`HITL request failed: ${response.error.message}`);
        }
        
        return response.result;
    }
    
    _sendRequest(request) {
        return new Promise((resolve, reject) => {
            this.pendingRequests.set(request.id, { resolve, reject });
            this._writeMessage(request);
            
            // Timeout after 5 minutes
            setTimeout(() => {
                if (this.pendingRequests.has(request.id)) {
                    this.pendingRequests.delete(request.id);
                    reject(new Error('Request timeout'));
                }
            }, 300000);
        });
    }
    
    _sendNotification(notification) {
        this._writeMessage(notification);
    }
    
    _writeMessage(message) {
        const data = JSON.stringify(message) + '\n';
        this.process.stdin.write(data);
    }
    
    _handleMessage(message) {
        if (message.id && this.pendingRequests.has(message.id)) {
            const { resolve } = this.pendingRequests.get(message.id);
            this.pendingRequests.delete(message.id);
            resolve(message);
        }
    }
    
    _nextId() {
        return ++this.requestId;
    }
    
    close() {
        if (this.process) {
            this.process.kill();
        }
    }
}

// Usage Example
async function main() {
    const client = new LoopgateMCPClient();
    await client.connect();
    
    await client.initialize('TradingBot', '1.0.0');
    
    const result = await client.requestApproval(
        'trading-session-1',
        'algorithmic-trader',
        'Execute large trade: Buy 10,000 AAPL at $150.25?',
        ['Execute', 'Cancel', 'Reduce Size'],
        { symbol: 'AAPL', value: '$1,502,500', risk: 'Medium' }
    );
    
    console.log('Trade approval result:', result);
    client.close();
}

main().catch(console.error);
```

### Go MCP Client (Using Built-in SDK)

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
    client := client.NewMCPClient()
    
    // Connect to Loopgate MCP server
    if err := client.ConnectToServer("./build/loopgate-mcp"); err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Initialize the connection
    if err := client.Initialize("ContentModerator", "1.0.0"); err != nil {
        log.Fatal(err)
    }
    
    // Send HITL request for content moderation
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    request := client.HITLRequest{
        SessionID: "moderation-session-1",
        ClientID:  "content-ai",
        Message:   "Flag this content as inappropriate? AI confidence: 75%",
        Options:   []string{"Flag", "Approve", "Escalate"},
        Metadata: map[string]interface{}{
            "content_type": "image",
            "ai_confidence": 0.75,
            "user_reports": 3,
        },
    }
    
    response, err := client.SendHITLRequest(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    
    if response.Approved {
        log.Println("Content approved for publication")
    } else {
        log.Printf("Content flagged: %s", response.Response)
    }
}
```

## Integration with AI Frameworks

### Claude Desktop/API with MCP

Add to your Claude MCP configuration:

```json
{
  "mcpServers": {
    "loopgate": {
      "command": "./build/loopgate-mcp",
      "env": {
        "TELEGRAM_BOT_TOKEN": "your_bot_token_here"
      }
    }
  }
}
```

### OpenAI Assistants API

```python
import openai
from loopgate_mcp_client import LoopgateMCPClient

# Function for OpenAI to call
async def request_human_approval(message: str, options: list = None) -> dict:
    client = LoopgateMCPClient()
    await client.connect()
    await client.initialize("OpenAI-Assistant", "1.0.0")
    
    result = await client.request_approval(
        session_id="openai-session-1",
        client_id="assistant",
        message=message,
        options=options
    )
    
    await client.close()
    return result

# Register with OpenAI
tools = [{
    "type": "function",
    "function": {
        "name": "request_human_approval",
        "description": "Request human approval for sensitive actions",
        "parameters": {
            "type": "object",
            "properties": {
                "message": {"type": "string"},
                "options": {"type": "array", "items": {"type": "string"}}
            },
            "required": ["message"]
        }
    }
}]
```

## Best Practices

### 1. Session Management
- Use descriptive session IDs that identify your AI agent
- Register sessions before sending HITL requests
- Clean up inactive sessions periodically

### 2. Error Handling
- Always handle MCP errors gracefully
- Implement timeouts for HITL requests
- Retry logic for transient failures

### 3. Message Design
- Write clear, actionable messages for humans
- Provide context in metadata
- Use appropriate option sets when applicable

### 4. Performance
- Reuse MCP connections when possible
- Implement connection pooling for high-volume scenarios
- Monitor response times and success rates

## Troubleshooting

### Common Issues

1. **Connection Failed**: Ensure Loopgate MCP server is running and accessible
2. **Tool Not Found**: Verify server initialization completed successfully  
3. **Session Not Found**: Register session before sending HITL requests
4. **Timeout Errors**: Check if human operator responded within timeout period

### Debug Mode

Enable debug logging:
```bash
LOG_LEVEL=debug ./build/loopgate-mcp
```

### Testing MCP Integration

```bash
# Test MCP server manually
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2.0","clientInfo":{"name":"test","version":"1.0"},"capabilities":{}}}' | ./build/loopgate-mcp
```