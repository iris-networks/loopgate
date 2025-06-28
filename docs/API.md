# 📡 API Reference

## MCP Protocol Tools

Loopgate exposes MCP tools for seamless AI agent integration:

### `request_human_input`
Send human-in-the-loop requests via MCP protocol.

```json
{
  "name": "request_human_input",
  "arguments": {
    "session_id": "my-agent-session",
    "client_id": "my-ai-agent",
    "message": "Should I proceed with this action?",
    "options": ["Yes", "No", "Maybe"],
    "metadata": {"key": "value"}
  }
}
```

### `check_request_status`
Check the status of a pending request.

```json
{
  "name": "check_request_status",
  "arguments": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

### `list_pending_requests`
List all pending HITL requests.

```json
{
  "name": "list_pending_requests",
  "arguments": {}
}
```

### `cancel_request`
Cancel a pending request.

```json
{
  "name": "cancel_request",
  "arguments": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

## HTTP Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/hitl/request` | POST | Submit HITL request (returns immediately) |
| `/hitl/poll` | GET | Poll for request status and response |
| `/hitl/register` | POST | Register AI agent session |
| `/hitl/status` | GET | Check session status |
| `/hitl/deactivate` | POST | Deactivate session |
| `/hitl/pending` | GET | List pending requests |
| `/hitl/cancel` | POST | Cancel pending request |
| `/health` | GET | Server health check |
| `/mcp` | POST | MCP protocol endpoint |
| `/mcp/tools` | GET | List available MCP tools |
| `/mcp/capabilities` | GET | Get MCP server capabilities |