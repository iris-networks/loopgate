# 📡 API Reference

## Overview

Loopgate provides a set of APIs for managing Human-in-the-Loop (HITL) workflows, MCP protocol interactions, and user/API key management for SaaS applications.

Authentication is handled via two mechanisms:
1.  **JWT Bearer Tokens**: For user authentication and managing API keys. Obtained via the `/api/auth/login` endpoint.
2.  **API Keys**: For authorizing access to protected service APIs (e.g., new SaaS APIs, or potentially existing MCP/HITL endpoints if configured).

## Authentication Endpoints

These endpoints are used for user registration and login to obtain a JWT for managing API keys.

### Register User

*   **Endpoint**: `POST /api/auth/register`
*   **Description**: Creates a new user account.
*   **Request Body**: `application/json`
    ```json
    {
      "username": "your_username",
      "password": "your_password"
    }
    ```
    *   `username` (string, required): Desired username.
    *   `password` (string, required): Desired password (min 8 characters).
*   **Success Response (201 Created)**: `application/json`
    ```json
    {
      "message": "User registered successfully",
      "user_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    }
    ```
*   **Error Responses**:
    *   `400 Bad Request`: Invalid payload, empty fields, password too short.
    *   `409 Conflict`: Username already exists.
    *   `500 Internal Server Error`: Server-side issue.

### Login User

*   **Endpoint**: `POST /api/auth/login`
*   **Description**: Authenticates a user and returns a JWT.
*   **Request Body**: `application/json`
    ```json
    {
      "username": "your_username",
      "password": "your_password"
    }
    ```
*   **Success Response (200 OK)**: `application/json`
    ```json
    {
      "token": "your_jwt_token_string",
      "user_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "username": "your_username"
    }
    ```
    *   `token`: The JWT Bearer token. Use this in the `Authorization` header for API key management endpoints.
*   **Error Responses**:
    *   `400 Bad Request`: Invalid payload, empty fields.
    *   `401 Unauthorized`: Invalid username or password.
    *   `500 Internal Server Error`: Server-side issue.

## API Key Management Endpoints

These endpoints require JWT Bearer token authentication. Include the token in the `Authorization` header: `Authorization: Bearer <your_jwt_token>`.

### Create API Key

*   **Endpoint**: `POST /api/user/apikeys`
*   **Description**: Creates a new API key for the authenticated user.
*   **Request Body**: `application/json` (optional)
    ```json
    {
      "label": "My production key",
      "expires_at": "2025-12-31T23:59:59Z"
    }
    ```
    *   `label` (string, optional): A descriptive label for the key.
    *   `expires_at` (string, optional): Expiration date in RFC3339 format. If omitted, the key does not expire.
*   **Success Response (201 Created)**: `application/json`
    ```json
    {
      "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "raw_key": "lk_pub_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "label": "My production key",
      "prefix": "lk_pub_",
      "expires_at": "2025-12-31T23:59:59Z",
      "created_at": "2024-01-01T12:00:00Z"
    }
    ```
    *   **Important**: The `raw_key` is displayed **only once** upon creation. Store it securely.
*   **Error Responses**:
    *   `400 Bad Request`: Invalid `expires_at` format.
    *   `401 Unauthorized`: JWT token missing or invalid.
    *   `500 Internal Server Error`.

### List API Keys

*   **Endpoint**: `GET /api/user/apikeys`
*   **Description**: Lists all API keys for the authenticated user.
*   **Success Response (200 OK)**: `application/json`
    ```json
    [
      {
        "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
        "label": "My production key",
        "prefix": "lk_pub_",
        "last_used_at": "2024-01-10T10:00:00Z",
        "expires_at": "2025-12-31T23:59:59Z",
        "created_at": "2024-01-01T12:00:00Z",
        "is_active": true
      }
      // ... more keys
    ]
    ```
*   **Error Responses**:
    *   `401 Unauthorized`: JWT token missing or invalid.
    *   `500 Internal Server Error`.

### Revoke API Key

*   **Endpoint**: `DELETE /api/user/apikeys/{key_id}`
*   **Description**: Revokes (deactivates) an API key.
*   **Path Parameter**:
    *   `key_id` (UUID string, required): The ID of the API key to revoke.
*   **Success Response (200 OK)**: `application/json`
    ```json
    {
      "message": "API key revoked successfully"
    }
    ```
*   **Error Responses**:
    *   `400 Bad Request`: Invalid `key_id` format.
    *   `401 Unauthorized`: JWT token missing or invalid.
    *   `404 Not Found`: API key not found or not owned by the user.
    *   `500 Internal Server Error`.

## Using API Keys for Service Access

To access API key protected endpoints (e.g., specific SaaS APIs, or potentially MCP/HITL services if configured for API key auth), include your generated API key in the request headers:

Option 1 (Recommended): `Authorization` Header
```
Authorization: Bearer <YOUR_API_KEY>
```
Example: `Authorization: Bearer lk_pub_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

Option 2: `X-API-Key` Header
```
X-API-Key: <YOUR_API_KEY>
```
Example: `X-API-Key: lk_pub_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

*(Note: The specific Loopgate endpoints protected by API keys will be detailed by your service administrator or in the relevant service documentation. For example, new SaaS APIs might be under `/api/saas/*` and require API key authentication.)*

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