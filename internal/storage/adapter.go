package storage

import (
	"loopgate/internal/types"
	// "time" // Removed unused import
)

// StorageAdapter defines the interface for data persistence.
type StorageAdapter interface {
	RegisterSession(sessionID, clientID string, telegramID int64) error
	DeactivateSession(sessionID string) error
	GetSession(sessionID string) (*types.Session, error)
	GetTelegramID(clientID string) (int64, error)
	StoreRequest(request *types.HITLRequest) error
	GetRequest(requestID string) (*types.HITLRequest, error)
	UpdateRequestResponse(requestID, response string, approved bool) error
	GetPendingRequests() ([]*types.HITLRequest, error)
	CancelRequest(requestID string) error
	GetActiveSessions() ([]*types.Session, error)

	// Add any other methods needed for data persistence, for example:
	// GetSessionByClientID(clientID string) (*types.Session, error)
	// GetRequestsBySessionID(sessionID string) ([]*types.HITLRequest, error)
	// DeleteExpiredRequests(olderThan time.Time) error
}
