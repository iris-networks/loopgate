package storage

import (
	"errors"
	"loopgate/internal/types"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SQLiteStorageAdapter implements the StorageAdapter interface for SQLite.
type SQLiteStorageAdapter struct {
	db *gorm.DB
}

// NewSQLiteStorageAdapter creates a new SQLiteStorageAdapter.
// It will also automatically migrate the schema.
// For in-memory SQLite, use "file::memory:?cache=shared" as the dsn.
// For a file-based SQLite, use the file path "your_database_name.db".
func NewSQLiteStorageAdapter(dsn string) (*SQLiteStorageAdapter, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate schema
	// GORM will create these tables if they don't exist.
	// The types.Session, types.HITLRequest, types.User, and types.APIKey structs
	// should be compatible with SQLite if they are with PostgreSQL,
	// as GORM abstracts SQL differences.
	err = db.AutoMigrate(&types.Session{}, &types.HITLRequest{}, &types.User{}, &types.APIKey{})
	if err != nil {
		// Attempt to close connection if migration fails
		sqlDB, _ := db.DB()
		sqlDB.Close()
		return nil, err
	}

	return &SQLiteStorageAdapter{db: db}, nil
}

// RegisterSession stores a new session.
func (s *SQLiteStorageAdapter) RegisterSession(sessionID, clientID string, telegramID int64) error {
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
func (s *SQLiteStorageAdapter) DeactivateSession(sessionID string) error {
	return s.db.Model(&types.Session{}).Where("id = ?", sessionID).Update("active", false).Error
}

// GetSession retrieves a session by its ID.
func (s *SQLiteStorageAdapter) GetSession(sessionID string) (*types.Session, error) {
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
func (s *SQLiteStorageAdapter) GetTelegramID(clientID string) (int64, error) {
	var session types.Session
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
func (s *SQLiteStorageAdapter) StoreRequest(request *types.HITLRequest) error {
	return s.db.Create(request).Error
}

// GetRequest retrieves a HITL request by its ID.
func (s *SQLiteStorageAdapter) GetRequest(requestID string) (*types.HITLRequest, error) {
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
func (s *SQLiteStorageAdapter) UpdateRequestResponse(requestID, response string, approved bool) error {
	// GORM's Save method will update all fields of the struct if it has a primary key.
	// First, retrieve the request to ensure it exists and to have its current state.
	tx := s.db.Begin()
	var request types.HITLRequest
	err := tx.First(&request, "id = ?", requestID).Error
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("request not found")
		}
		return err
	}

	now := time.Now()
	request.Response = response
	request.Approved = approved
	request.Status = types.RequestStatusCompleted
	request.RespondedAt = &now

	if err := tx.Save(&request).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// GetPendingRequests retrieves all requests with a 'pending' status.
func (s *SQLiteStorageAdapter) GetPendingRequests() ([]*types.HITLRequest, error) {
	var pendingRequests []*types.HITLRequest
	err := s.db.Where("status = ?", types.RequestStatusPending).Find(&pendingRequests).Error
	if err != nil {
		return nil, err
	}
	return pendingRequests, nil
}

// CancelRequest marks a request as 'canceled'.
func (s *SQLiteStorageAdapter) CancelRequest(requestID string) error {
	return s.db.Model(&types.HITLRequest{}).Where("id = ?", requestID).Update("status", types.RequestStatusCanceled).Error
}

// GetActiveSessions retrieves all sessions that are currently active.
func (s *SQLiteStorageAdapter) GetActiveSessions() ([]*types.Session, error) {
	var activeSessions []*types.Session
	err := s.db.Where("active = ?", true).Find(&activeSessions).Error
	if err != nil {
		return nil, err
	}
	return activeSessions, nil
}

// Close closes the database connection.
// For SQLite, especially in-memory, this might not be strictly necessary
// but good practice for consistency and if file-based DBs are used.
func (s *SQLiteStorageAdapter) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// --- User management methods ---

// CreateUser creates a new user.
func (s *SQLiteStorageAdapter) CreateUser(user *types.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	return s.db.Create(user).Error
}

// GetUserByUsername retrieves a user by their username.
func (s *SQLiteStorageAdapter) GetUserByUsername(username string) (*types.User, error) {
	var user types.User
	err := s.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by their ID.
func (s *SQLiteStorageAdapter) GetUserByID(userID uuid.UUID) (*types.User, error) {
	var user types.User
	err := s.db.First(&user, "id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// --- APIKey management methods ---

// CreateAPIKey creates a new API key.
func (s *SQLiteStorageAdapter) CreateAPIKey(apiKey *types.APIKey) error {
	if apiKey.ID == uuid.Nil {
		apiKey.ID = uuid.New()
	}
	apiKey.CreatedAt = time.Now()
	apiKey.IsActive = true
	return s.db.Create(apiKey).Error
}

// GetAPIKeyByHash retrieves an API key by its hash.
func (s *SQLiteStorageAdapter) GetAPIKeyByHash(keyHash string) (*types.APIKey, error) {
	var apiKey types.APIKey
	err := s.db.Where("key_hash = ?", keyHash).First(&apiKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("api key not found")
		}
		return nil, err
	}
	return &apiKey, nil
}

// GetActiveAPIKeyByHash retrieves an active API key by its hash.
func (s *SQLiteStorageAdapter) GetActiveAPIKeyByHash(keyHash string) (*types.APIKey, error) {
	var apiKey types.APIKey
	err := s.db.Where("key_hash = ? AND is_active = ?", keyHash, true).First(&apiKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("active api key not found")
		}
		return nil, err
	}
	return &apiKey, nil
}

// GetAPIKeysByUserID retrieves all API keys for a given user ID.
func (s *SQLiteStorageAdapter) GetAPIKeysByUserID(userID uuid.UUID) ([]*types.APIKey, error) {
	var apiKeys []*types.APIKey
	err := s.db.Where("user_id = ?", userID).Find(&apiKeys).Error
	if err != nil {
		return nil, err
	}
	return apiKeys, nil
}

// RevokeAPIKey marks an API key as inactive. It ensures the key belongs to the user.
func (s *SQLiteStorageAdapter) RevokeAPIKey(apiKeyID uuid.UUID, userID uuid.UUID) error {
	var apiKey types.APIKey
	err := s.db.First(&apiKey, "id = ? AND user_id = ?", apiKeyID, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("api key not found or not owned by user")
		}
		return err
	}
	return s.db.Model(&apiKey).Update("is_active", false).Error
}

// UpdateAPIKeyLastUsed updates the last used timestamp for an API key.
func (s *SQLiteStorageAdapter) UpdateAPIKeyLastUsed(apiKeyID uuid.UUID) error {
	now := time.Now()
	return s.db.Model(&types.APIKey{}).Where("id = ?", apiKeyID).Update("last_used_at", &now).Error
}
