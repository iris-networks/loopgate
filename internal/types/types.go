package types

import (
	"time"
)

type RequestType string

const (
	RequestTypeConfirmation RequestType = "confirmation"
	RequestTypeInput        RequestType = "input"
	RequestTypeChoice       RequestType = "choice"
)

type RequestStatus string

const (
	RequestStatusPending   RequestStatus = "pending"
	RequestStatusCompleted RequestStatus = "completed"
	RequestStatusTimeout   RequestStatus = "timeout"
	RequestStatusCanceled  RequestStatus = "canceled"
)

type HITLRequest struct {
	ID            string                 `json:"id" gorm:"primaryKey"`
	SessionID     string                 `json:"session_id"`
	ClientID      string                 `json:"client_id"`
	Message       string                 `json:"message"`
	RequestType   RequestType            `json:"request_type"`
	Options       []string               `json:"options,omitempty" gorm:"serializer:json"`
	Timeout       int                    `json:"timeout_seconds"`
	CallbackURL   string                 `json:"callback_url,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" gorm:"serializer:json"`
	Status        RequestStatus          `json:"status"`
	Response      string                 `json:"response,omitempty"`
	Approved      bool                   `json:"approved"`
	CreatedAt     time.Time              `json:"created_at"`
	RespondedAt   *time.Time             `json:"responded_at,omitempty"`
	TelegramMsgID int                    `json:"telegram_msg_id,omitempty"`
}

type Session struct {
	ID         string `json:"id" gorm:"primaryKey"`
	ClientID   string `json:"client_id"`
	TelegramID int64  `json:"telegram_id"`
	Active     bool   `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
}

type HITLResponse struct {
	RequestID string    `json:"request_id"`
	Status    RequestStatus `json:"status"`
	Response  string    `json:"response,omitempty"`
	Approved  bool      `json:"approved"`
	Timestamp time.Time `json:"timestamp"`
}

type SessionRegistration struct {
	SessionID  string `json:"session_id"`
	ClientID   string `json:"client_id"`
	TelegramID int64  `json:"telegram_id"`
}

type PollResponse struct {
	Status      RequestStatus `json:"status"`
	Response    string        `json:"response,omitempty"`
	Approved    bool          `json:"approved"`
	RequestID   string        `json:"request_id"`
	Completed   bool          `json:"completed"`
}

type MCPRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
	ID     interface{} `json:"id"`
}

type MCPResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  *MCPError   `json:"error,omitempty"`
	ID     interface{} `json:"id"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type MCPCapabilities struct {
	Tools map[string]interface{} `json:"tools,omitempty"`
}

type MCPInitializeParams struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    MCPCapabilities `json:"capabilities"`
	ClientInfo      MCPClientInfo   `json:"clientInfo"`
}

type MCPClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type MCPServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type MCPInitializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    MCPCapabilities `json:"capabilities"`
	ServerInfo      MCPServerInfo   `json:"serverInfo"`
}