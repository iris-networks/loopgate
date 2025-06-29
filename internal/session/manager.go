package session

import (
	"errors"
	"loopgate/internal/store"
	"loopgate/internal/types"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Manager handles business logic related to sessions and HITL requests.
// It now uses a MongoDB backend for data persistence.
type Manager struct {
	db *mongo.Database
}

// NewManager creates a new session manager with a MongoDB database connection.
func NewManager(db *mongo.Database) *Manager {
	if db == nil {
		panic("session.NewManager: mongo.Database instance cannot be nil")
	}
	return &Manager{
		db: db,
	}
}

// RegisterSession creates a new session and stores it in MongoDB.
func (m *Manager) RegisterSession(sessionID, clientID string, telegramID int64) error {
	session := &types.Session{
		ID:         sessionID,
		ClientID:   clientID,
		TelegramID: telegramID,
		Active:     true,
		CreatedAt:  time.Now(),
	}
	// TODO: Decide on handling if sessionID already exists.
	// store.MongoRegisterSession currently returns an error if _id (session.ID) conflicts.
	// Consider an upsert in MongoRegisterSession or specific error handling here.
	err := store.MongoRegisterSession(m.db, session)
	if err != nil {
		// Potentially check for mongo.IsDuplicateKeyError(err) if specific handling is needed
		return err
	}
	return nil
}

// DeactivateSession marks a session as inactive in MongoDB.
func (m *Manager) DeactivateSession(sessionID string) error {
	err := store.MongoDeactivateSession(m.db, sessionID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return errors.New("session not found")
	}
	return err
}

// GetSession retrieves a session by its ID from MongoDB.
func (m *Manager) GetSession(sessionID string) (*types.Session, error) {
	session, err := store.MongoGetSession(m.db, sessionID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("session not found")
		}
		return nil, err
	}
	if session == nil { // Should be covered by ErrNoDocuments, but as a safeguard
		return nil, errors.New("session not found")
	}
	return session, nil
}

// GetTelegramID retrieves the Telegram ID for an active client session from MongoDB.
func (m *Manager) GetTelegramID(clientID string) (int64, error) {
	telegramID, err := store.MongoGetTelegramIDForClient(m.db, clientID)
	if err != nil {
		// The store function store.MongoGetTelegramIDForClient already returns a specific error
		// (e.g., errors.New("active session not found for client")) when no document is found.
		// So, we can often return err directly.
		// If we wanted to standardize to a generic "client not found" vs "other db error",
		// we might check 'errors.Is(err, store.ErrNotFound)' or similar if store exposed typed errors.
		return 0, err
	}
	return telegramID, nil
}

// StoreRequest persists a new HITL request to MongoDB.
func (m *Manager) StoreRequest(request *types.HITLRequest) error {
	// The original StoreRequest didn't return an error, but DAL operations do.
	// It's better to propagate the error.
	return store.MongoStoreRequest(m.db, request)
}

// GetRequest retrieves a HITL request by its ID from MongoDB.
func (m *Manager) GetRequest(requestID string) (*types.HITLRequest, error) {
	request, err := store.MongoGetRequest(m.db, requestID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("request not found")
		}
		return nil, err
	}
	if request == nil { // Safeguard
		return nil, errors.New("request not found")
	}
	return request, nil
}

// UpdateRequestResponse updates a HITL request with the response details in MongoDB.
func (m *Manager) UpdateRequestResponse(requestID, response string, approved bool) error {
	status := types.RequestStatusCompleted
	// Note: The original implementation didn't explicitly handle other statuses like Timeout here.
	// This function is specifically for when a human responds.

	err := store.MongoUpdateRequestResponse(m.db, requestID, response, approved, status, time.Now())
	if errors.Is(err, mongo.ErrNoDocuments) {
		return errors.New("request not found when updating response")
	}
	return err
}

// GetPendingRequests retrieves all pending HITL requests from MongoDB.
// The new DAL function store.MongoGetPendingRequests also doesn't take clientID.
func (m *Manager) GetPendingRequests() ([]*types.HITLRequest, error) {
	requests, err := store.MongoGetPendingRequests(m.db)
	if err != nil {
		return nil, err
	}
	return requests, nil
}

// CancelRequest marks a HITL request as canceled in MongoDB.
func (m *Manager) CancelRequest(requestID string) error {
	err := store.MongoCancelRequest(m.db, requestID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return errors.New("request not found when canceling")
	}
	return err
}

// GetActiveSessions retrieves all active sessions from MongoDB.
func (m *Manager) GetActiveSessions() ([]*types.Session, error) {
	sessions, err := store.MongoGetActiveSessions(m.db)
	if err != nil {
		return nil, err
	}
	return sessions, nil
}