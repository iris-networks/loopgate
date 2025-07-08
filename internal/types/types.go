package types

import (
	"time"

	"github.com/google/uuid"
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

// User represents a user account in the system.
type User struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;"`
	Username     string    `json:"username" gorm:"uniqueIndex;not null;size:255"`
	PasswordHash string    `json:"-" gorm:"not null"` // Avoid exposing password hash in JSON
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// APIKey represents an API key associated with a user.
type APIKey struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;"`
	KeyHash    string     `json:"-" gorm:"uniqueIndex;not null"` // Store hash of the key, not the key itself
	UserID     uuid.UUID  `json:"user_id" gorm:"type:uuid;not null"`
	User       User       `json:"-" gorm:"foreignKey:UserID;references:ID"` // GORM relation
	Label      string     `json:"label" gorm:"size:255"`
	Prefix     string     `json:"prefix" gorm:"size:10;not null"` // e.g., "lk_pub_" for quick identification
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	IsActive   bool       `json:"is_active" gorm:"default:true;not null"`
}

// Claims represents the JWT claims, embedding jwt.RegisteredClaims for standard fields.
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	// In the actual JWT implementation, we'll embed jwt.RegisteredClaims
	// For now, this type definition is a placeholder for structure.
	// e.g. StandardClaims jwt.RegisteredClaims `json:"standard_claims"`
	RegisteredClaims interface{} `json:"registered_claims,omitempty"`
}