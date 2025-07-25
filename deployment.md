# 🚀 Production Deployment

### Prerequisites
- Go 1.21+
- Telegram Bot Token (create via @BotFather)
- Server with public internet access

### Deployment Options

#### Traditional Server
```bash
# Build for production
make build

# Create systemd service
sudo tee /etc/systemd/system/loopgate.service > /dev/null <<EOF
[Unit]
Description=Loopgate MCP Server
After=network.target

[Service]
Type=simple
User=loopgate
WorkingDirectory=/opt/loopgate
ExecStart=/opt/loopgate/loopgate
Environment=TELEGRAM_BOT_TOKEN=your_token_here
Environment=SERVER_PORT=8080
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl enable loopgate
sudo systemctl start loopgate
```

#### Docker Compose
```yaml
version: '3.8'
services:
  loopgate:
    build: .
    ports:
      - "8080:8080"
    environment:
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - LOG_LEVEL=info
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

#### Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: loopgate
  labels:
    app: loopgate
spec:
  replicas: 2
  selector:
    matchLabels:
      app: loopgate
  template:
    metadata:
      labels:
        app: loopgate
    spec:
      containers:
      - name: loopgate
        image: loopgate:latest
        ports:
        - containerPort: 8080
        env:
        - name: TELEGRAM_BOT_TOKEN
          valueFrom:
            secretKeyRef:
              name: loopgate-secret
              key: telegram-token
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: loopgate-service
spec:
  selector:
    app: loopgate
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

## 📊 Monitoring and Observability

### Health Checks
```bash
# Basic health check
curl http://localhost:8080/health

# Detailed session status
curl "http://localhost:8080/hitl/status?session_id=my-session"

# List pending requests
curl http://localhost:8080/hitl/pending
```

### Logging
Loopgate provides structured logging for monitoring:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "message": "HITL request submitted",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "client_id": "production-bot",
  "session_id": "deploy-session"
}
```

### Metrics Integration
Future versions will include Prometheus metrics:

```
# HELP loopgate_hitl_requests_total Total HITL requests
# TYPE loopgate_hitl_requests_total counter
loopgate_hitl_requests_total{client_id="production-bot"} 1234

# HELP loopgate_active_sessions Current active sessions  
# TYPE loopgate_active_sessions gauge
loopgate_active_sessions 5

# HELP loopgate_response_time_seconds HITL response time
# TYPE loopgate_response_time_seconds histogram
```

## 🔒 Security Considerations

### Authentication
- Telegram bot tokens should be kept secure
- Consider implementing API key authentication for HTTP endpoints
- Use HTTPS in production environments

### Network Security
```bash
# Firewall configuration (Ubuntu/Debian)
sudo ufw allow 22    # SSH
sudo ufw allow 8080  # Loopgate
sudo ufw enable
```

### Environment Variables
```bash
# Use secrets management in production
export TELEGRAM_BOT_TOKEN=$(vault kv get -field=token secret/loopgate)
```