package storage

import (
	"errors"
	"loopgate/internal/types"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgreSQLStorageAdapter implements the StorageAdapter interface for PostgreSQL.
type PostgreSQLStorageAdapter struct {
	db *gorm.DB
}

// NewPostgreSQLStorageAdapter creates a new PostgreSQLStorageAdapter.
// It will also automatically migrate the schema.
func NewPostgreSQLStorageAdapter(dsn string) (*PostgreSQLStorageAdapter, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate schema
	err = db.AutoMigrate(&types.Session{}, &types.HITLRequest{})
	if err != nil {
		// Attempt to close connection if migration fails
		sqlDB, _ := db.DB()
		sqlDB.Close()
		return nil, err
	}

	return &PostgreSQLStorageAdapter{db: db}, nil
}

// RegisterSession stores a new session.
func (s *PostgreSQLStorageAdapter) RegisterSession(sessionID, clientID string, telegramID int64) error {
	session := &types.Session{
		ID:         sessionID,
		ClientID:   clientID,
		TelegramID: telegramID,
		Active:     true,
		CreatedAt:  time.Now(),
	}
	return s.db.Create(session).Error
}

// DeactivateSession marks a session as inactive.
func (s *PostgreSQLStorageAdapter) DeactivateSession(sessionID string) error {
	return s.db.Model(&types.Session{}).Where("id = ?", sessionID).Update("active", false).Error
}

// GetSession retrieves a session by its ID.
func (s *PostgreSQLStorageAdapter) GetSession(sessionID string) (*types.Session, error) {
	var session types.Session
	err := s.db.First(&session, "id = ?", sessionID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("session not found")
		}
		return nil, err
	}
	return &session, nil
}

// GetTelegramID retrieves the Telegram ID associated with an active Client ID.
func (s *PostgreSQLStorageAdapter) GetTelegramID(clientID string) (int64, error) {
	var session types.Session
	// Assuming a clientID can only have one active session at a time.
	// If not, this logic might need adjustment or clarification.
	err := s.db.Where("client_id = ? AND active = ?", clientID, true).First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("active session for client not found")
		}
		return 0, err
	}
	return session.TelegramID, nil
}

// StoreRequest stores a new HITL request.
func (s *PostgreSQLStorageAdapter) StoreRequest(request *types.HITLRequest) error {
	return s.db.Create(request).Error
}

// GetRequest retrieves a HITL request by its ID.
func (s *PostgreSQLStorageAdapter) GetRequest(requestID string) (*types.HITLRequest, error) {
	var request types.HITLRequest
	err := s.db.First(&request, "id = ?", requestID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("request not found")
		}
		return nil, err
	}
	return &request, nil
}

// UpdateRequestResponse updates the response and status of a HITL request.
func (s *PostgreSQLStorageAdapter) UpdateRequestResponse(requestID, response string, approved bool) error {
	request, err := s.GetRequest(requestID)
	if err != nil {
		return err
	}

	now := time.Now()
	request.Response = response
	request.Approved = approved
	request.Status = types.RequestStatusCompleted
	request.RespondedAt = &now

	return s.db.Save(request).Error
}

// GetPendingRequests retrieves all requests with a 'pending' status.
func (s *PostgreSQLStorageAdapter) GetPendingRequests() ([]*types.HITLRequest, error) {
	var pendingRequests []*types.HITLRequest
	err := s.db.Where("status = ?", types.RequestStatusPending).Find(&pendingRequests).Error
	if err != nil {
		return nil, err
	}
	return pendingRequests, nil
}

// CancelRequest marks a request as 'canceled'.
func (s *PostgreSQLStorageAdapter) CancelRequest(requestID string) error {
	return s.db.Model(&types.HITLRequest{}).Where("id = ?", requestID).Update("status", types.RequestStatusCanceled).Error
}

// GetActiveSessions retrieves all sessions that are currently active.
func (s *PostgreSQLStorageAdapter) GetActiveSessions() ([]*types.Session, error) {
	var activeSessions []*types.Session
	err := s.db.Where("active = ?", true).Find(&activeSessions).Error
	if err != nil {
		return nil, err
	}
	return activeSessions, nil
}

// Close closes the database connection.
func (s *PostgreSQLStorageAdapter) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
