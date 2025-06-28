# Loopgate Usage Guide

This guide provides detailed usage examples and best practices for Loopgate.

## Getting Started

### 1. Installation and Setup

```bash
# Clone and build
git clone https://github.com/your-username/loopgate.git
cd loopgate
make build

# Set up environment
export TELEGRAM_BOT_TOKEN="your_bot_token"
export SERVER_PORT=8080
```

### 2. Create Telegram Bot

1. Message @BotFather on Telegram
2. Send `/newbot` command
3. Follow instructions to get bot token
4. Set token in environment variables

### 3. Get Your Telegram ID

Send a message to your bot, then check:
```bash
curl "https://api.telegram.org/bot{YOUR_BOT_TOKEN}/getUpdates"
```

Look for `"from": {"id": 123456789}` in the response.

## Basic Usage Patterns

### 1. Simple Approval Request

```python
import requests

# Register session
requests.post('http://localhost:8080/hitl/register', json={
    "session_id": "approval-bot",
    "client_id": "my-ai",
    "telegram_id": 123456789
})

# Request approval
response = requests.post('http://localhost:8080/hitl/request', json={
    "session_id": "approval-bot", 
    "client_id": "my-ai",
    "message": "Approve this action?",
    "options": ["Yes", "No"]
})

request_id = response.json()["request_id"]

# Poll for response
while True:
    poll = requests.get(f'http://localhost:8080/hitl/poll?request_id={request_id}')
    status = poll.json()
    if status["completed"]:
        print(f"Decision: {status['response']}")
        break
    time.sleep(2)
```

### 2. Input Collection

```python
# Request free-form input
response = requests.post('http://localhost:8080/hitl/request', json={
    "session_id": "input-bot",
    "client_id": "my-ai", 
    "message": "Please provide additional context:",
    "request_type": "input"
})
```

### 3. Multiple Choice

```python
# Present multiple options
response = requests.post('http://localhost:8080/hitl/request', json={
    "session_id": "choice-bot",
    "client_id": "my-ai",
    "message": "Select deployment environment:",
    "options": ["Development", "Staging", "Production"],
    "request_type": "choice"
})
```

## Advanced Patterns

### 1. Metadata and Context

```python
# Include rich metadata
response = requests.post('http://localhost:8080/hitl/request', json={
    "session_id": "deploy-bot",
    "client_id": "ci-cd",
    "message": "Deploy new version?",
    "options": ["Deploy", "Cancel", "Review"],
    "metadata": {
        "version": "v2.1.0",
        "tests_passed": 847,
        "code_coverage": "94.2%",
        "environment": "production",
        "commit_hash": "abc123",
        "author": "john.doe@company.com",
        "estimated_downtime": "2 minutes"
    }
})
```

### 2. Timeout Handling

```python
# Set custom timeout
response = requests.post('http://localhost:8080/hitl/request', json={
    "session_id": "urgent-bot",
    "client_id": "alert-system",
    "message": "Critical alert: High CPU usage detected",
    "options": ["Investigate", "Auto-scale", "Ignore"],
    "timeout_seconds": 300,  # 5 minutes
    "metadata": {
        "severity": "critical",
        "cpu_usage": "95%",
        "affected_servers": 12
    }
})
```

### 3. Multiple Sessions

```python
# Different sessions for different use cases
sessions = [
    {"session_id": "deploy", "client_id": "ci-cd", "telegram_id": 123456789},
    {"session_id": "alerts", "client_id": "monitoring", "telegram_id": 987654321},
    {"session_id": "approvals", "client_id": "finance", "telegram_id": 555666777}
]

for session in sessions:
    requests.post('http://localhost:8080/hitl/register', json=session)
```

## MCP Integration Patterns

### 1. Go MCP Client

```go
package main

import (
    "log"
    "loopgate/pkg/client"
)

func main() {
    client := client.NewMCPClient()
    defer client.Close()
    
    err := client.ConnectToServer("./loopgate")
    if err != nil {
        log.Fatal(err)
    }
    
    err = client.Initialize("ProductionAI", "1.0.0")
    if err != nil {
        log.Fatal(err)
    }
    
    // Request approval through MCP
    response, err := client.SendHITLRequest(
        "production-session",
        "deploy-ai",
        "Deploy critical security patch?",
        []string{"Deploy Immediately", "Schedule Maintenance", "Cancel"},
        map[string]interface{}{
            "patch_id": "SEC-2024-001",
            "severity": "critical",
            "affected_systems": []string{"auth", "payment", "user-data"},
        },
    )
    
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("MCP Response: %+v", response)
}
```

### 2. Tool Discovery

```go
// List available tools
tools, err := client.ListTools()
if err != nil {
    log.Fatal(err)
}

for _, tool := range tools {
    fmt.Printf("Tool: %s\n", tool.Name)
    fmt.Printf("Description: %s\n", tool.Description)
    fmt.Printf("Schema: %+v\n\n", tool.InputSchema)
}
```

## Workflow Examples

### 1. CI/CD Pipeline Integration

```python
class DeploymentApprover:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
        self.session_id = "cicd-pipeline"
        self.client_id = "github-actions"
        
    def request_deployment_approval(self, version, environment, test_results):
        message = f"Deploy {version} to {environment}?"
        
        metadata = {
            "version": version,
            "environment": environment,
            "tests_passed": test_results.get("passed", 0),
            "tests_failed": test_results.get("failed", 0),
            "code_coverage": test_results.get("coverage", "unknown"),
            "build_time": test_results.get("build_time"),
            "security_scan": test_results.get("security", "unknown")
        }
        
        options = ["Deploy", "Cancel"]
        if environment == "production":
            options.append("Deploy to Staging First")
            
        return self.send_request(message, options, metadata)
        
    def send_request(self, message, options, metadata):
        response = requests.post(f'{self.base_url}/hitl/request', json={
            "session_id": self.session_id,
            "client_id": self.client_id,
            "message": message,
            "options": options,
            "metadata": metadata
        })
        
        request_id = response.json()["request_id"]
        return self.wait_for_approval(request_id)
        
    def wait_for_approval(self, request_id):
        while True:
            poll_resp = requests.get(f'{self.base_url}/hitl/poll?request_id={request_id}')
            status = poll_resp.json()
            
            if status["completed"]:
                return {
                    "approved": status["approved"],
                    "response": status["response"],
                    "request_id": request_id
                }
                
            time.sleep(5)

# Usage in CI/CD
approver = DeploymentApprover()
result = approver.request_deployment_approval(
    version="v2.1.0",
    environment="production", 
    test_results={
        "passed": 847,
        "failed": 0,
        "coverage": "94.2%",
        "build_time": "3m 24s",
        "security": "passed"
    }
)

if result["approved"]:
    print("Deployment approved, proceeding...")
    # Execute deployment
else:
    print(f"Deployment denied: {result['response']}")
    # Handle rejection
```

### 2. Trading Bot Integration

```python
class TradingApprover:
    def __init__(self):
        self.session_id = "trading-bot"
        self.client_id = "algo-trader"
        
    def request_trade_approval(self, trade_details):
        risk_level = self.calculate_risk(trade_details)
        
        if risk_level == "low":
            return {"approved": True, "auto_approved": True}
            
        message = f"Execute {trade_details['action']} trade?"
        
        metadata = {
            "symbol": trade_details["symbol"],
            "quantity": trade_details["quantity"],
            "price": trade_details["price"],
            "total_value": trade_details["total_value"],
            "risk_level": risk_level,
            "market_conditions": trade_details.get("market_conditions"),
            "strategy": trade_details.get("strategy"),
            "confidence": trade_details.get("confidence")
        }
        
        options = ["Execute", "Cancel", "Reduce Size", "Wait for Better Price"]
        
        return self.send_request(message, options, metadata)
```

### 3. Content Moderation

```python
class ContentModerator:
    def __init__(self):
        self.session_id = "content-mod"
        self.client_id = "moderation-ai"
        
    def escalate_to_human(self, content, ai_analysis):
        confidence = ai_analysis.get("confidence", 0)
        
        if confidence > 0.95:
            return {"action": ai_analysis["recommendation"], "auto_moderated": True}
            
        message = f"Review flagged content (AI confidence: {confidence:.1%})"
        
        metadata = {
            "content_type": content["type"],
            "content_id": content["id"], 
            "ai_recommendation": ai_analysis["recommendation"],
            "ai_confidence": confidence,
            "flagged_categories": ai_analysis.get("categories", []),
            "user_id": content.get("user_id"),
            "reported_by": content.get("reported_by")
        }
        
        options = ["Approve", "Remove", "Warn User", "Escalate Further", "Request More Context"]
        
        return self.send_request(message, options, metadata)
```

## Error Handling

### 1. HTTP Errors

```python
try:
    response = requests.post('http://localhost:8080/hitl/request', json=data)
    response.raise_for_status()
    result = response.json()
except requests.exceptions.ConnectionError:
    print("Cannot connect to Loopgate server")
except requests.exceptions.Timeout:
    print("Request timed out")
except requests.exceptions.HTTPError as e:
    print(f"HTTP error: {e.response.status_code}")
except ValueError:
    print("Invalid JSON response")
```

### 2. Session Errors

```python
def ensure_session_active(session_id):
    try:
        response = requests.get(f'http://localhost:8080/hitl/status?session_id={session_id}')
        if response.status_code == 404:
            print(f"Session {session_id} not found, re-registering...")
            # Re-register session
            return False
        elif response.status_code == 200:
            session_data = response.json()
            if not session_data.get("active", False):
                print(f"Session {session_id} is inactive")
                return False
        return True
    except Exception as e:
        print(f"Error checking session: {e}")
        return False
```

### 3. Timeout Handling

```python
def poll_with_timeout(request_id, max_wait=300):
    start_time = time.time()
    
    while time.time() - start_time < max_wait:
        try:
            poll_resp = requests.get(f'http://localhost:8080/hitl/poll?request_id={request_id}')
            status = poll_resp.json()
            
            if status["completed"]:
                return status
                
        except Exception as e:
            print(f"Error polling: {e}")
            
        time.sleep(5)
        
    # Cancel request on timeout
    requests.post('http://localhost:8080/hitl/cancel', json={"request_id": request_id})
    raise TimeoutError(f"Request {request_id} timed out after {max_wait} seconds")
```

## Best Practices

1. **Session Management**
   - Use descriptive session IDs
   - Register sessions on startup
   - Check session status before requests

2. **Request Design**
   - Provide clear, concise messages
   - Include relevant context in metadata
   - Use appropriate timeouts

3. **Error Handling**
   - Handle network errors gracefully
   - Implement retry logic for transient failures
   - Log errors for debugging

4. **Performance**
   - Use connection pooling for high throughput
   - Implement efficient polling strategies
   - Cache session information

5. **Security**
   - Protect Telegram bot tokens
   - Validate input parameters
   - Use HTTPS in production