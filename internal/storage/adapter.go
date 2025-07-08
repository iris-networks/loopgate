package storage

import (
	"loopgate/internal/types"

	"github.com/google/uuid"
	// "time" // Removed unused import
)

// StorageAdapter defines the interface for data persistence.
type StorageAdapter interface {
	// Session and HITL methods (existing)
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

	// User management methods
	CreateUser(user *types.User) error
	GetUserByUsername(username string) (*types.User, error)
	GetUserByID(userID uuid.UUID) (*types.User, error)

	// APIKey management methods
	CreateAPIKey(apiKey *types.APIKey) error
	GetAPIKeyByHash(keyHash string) (*types.APIKey, error) // Primarily for checking uniqueness or internal lookup
	GetActiveAPIKeyByHash(keyHash string) (*types.APIKey, error) // For auth middleware, ensures key is active
	GetAPIKeysByUserID(userID uuid.UUID) ([]*types.APIKey, error)
	RevokeAPIKey(apiKeyID uuid.UUID, userID uuid.UUID) error // Confirms ownership via userID before revoking
	UpdateAPIKeyLastUsed(apiKeyID uuid.UUID) error

	// Add any other methods needed for data persistence, for example:
	// GetSessionByClientID(clientID string) (*types.Session, error)
	// GetRequestsBySessionID(sessionID string) ([]*types.HITLRequest, error)
	// DeleteExpiredRequests(olderThan time.Time) error
}
