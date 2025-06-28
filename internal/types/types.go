package types

import "time"

type HITLRequest struct {
	ID        string                 `json:"id"`
	ClientID  string                 `json:"client_id"`
	SessionID string                 `json:"session_id"`
	Message   string                 `json:"message"`
	Options   []string               `json:"options,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type HITLResponse struct {
	ID       string    `json:"id"`
	Response string    `json:"response"`
	Selected int       `json:"selected,omitempty"`
	Approved bool      `json:"approved"`
	Time     time.Time `json:"time"`
}

type Session struct {
	ID           string    `json:"id"`
	ClientID     string    `json:"client_id"`
	TelegramID   int64     `json:"telegram_id"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
}

type MCPMessage struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
	ID     string      `json:"id,omitempty"`
}