package storage

import (
	"errors"
	"loopgate/internal/types"
	"sync"
	"time"
)

// InMemoryStorageAdapter implements the StorageAdapter interface for in-memory storage.
type InMemoryStorageAdapter struct {
	sessions         map[string]*types.Session
	requests         map[string]*types.HITLRequest
	clientToTelegram map[string]int64
	mu               sync.RWMutex
}

// NewInMemoryStorageAdapter creates a new InMemoryStorageAdapter.
func NewInMemoryStorageAdapter() *InMemoryStorageAdapter {
	return &InMemoryStorageAdapter{
		sessions:         make(map[string]*types.Session),
		requests:         make(map[string]*types.HITLRequest),
		clientToTelegram: make(map[string]int64),
	}
}

// RegisterSession stores a new session.
func (s *InMemoryStorageAdapter) RegisterSession(sessionID, clientID string, telegramID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[sessionID]; exists {
		return errors.New("session already exists")
	}

	session := &types.Session{
		ID:         sessionID,
		ClientID:   clientID,
		TelegramID: telegramID,
		Active:     true,
		CreatedAt:  time.Now(),
	}

	s.sessions[sessionID] = session
	s.clientToTelegram[clientID] = telegramID
	return nil
}

// DeactivateSession marks a session as inactive.
func (s *InMemoryStorageAdapter) DeactivateSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return errors.New("session not found")
	}

	session.Active = false
	delete(s.clientToTelegram, session.ClientID) // Consider if ClientID should be removed or session just marked inactive
	return nil
}

// GetSession retrieves a session by its ID.
func (s *InMemoryStorageAdapter) GetSession(sessionID string) (*types.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, errors.New("session not found")
	}
	return session, nil
}

// GetTelegramID retrieves the Telegram ID associated with a Client ID.
func (s *InMemoryStorageAdapter) GetTelegramID(clientID string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	telegramID, exists := s.clientToTelegram[clientID]
	if !exists {
		return 0, errors.New("client not found")
	}
	return telegramID, nil
}

// StoreRequest stores a new HITL request.
func (s *InMemoryStorageAdapter) StoreRequest(request *types.HITLRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.requests[request.ID]; exists {
		return errors.New("request already exists")
	}
	s.requests[request.ID] = request
	return nil
}

// GetRequest retrieves a HITL request by its ID.
func (s *InMemoryStorageAdapter) GetRequest(requestID string) (*types.HITLRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	request, exists := s.requests[requestID]
	if !exists {
		return nil, errors.New("request not found")
	}
	return request, nil
}

// UpdateRequestResponse updates the response and status of a HITL request.
func (s *InMemoryStorageAdapter) UpdateRequestResponse(requestID, response string, approved bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	request, exists := s.requests[requestID]
	if !exists {
		return errors.New("request not found")
	}

	now := time.Now()
	request.Response = response
	request.Approved = approved
	request.Status = types.RequestStatusCompleted
	request.RespondedAt = &now
	return nil
}

// GetPendingRequests retrieves all requests with a 'pending' status.
func (s *InMemoryStorageAdapter) GetPendingRequests() ([]*types.HITLRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pending []*types.HITLRequest
	for _, request := range s.requests {
		if request.Status == types.RequestStatusPending {
			pending = append(pending, request)
		}
	}
	return pending, nil
}

// CancelRequest marks a request as 'canceled'.
func (s *InMemoryStorageAdapter) CancelRequest(requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	request, exists := s.requests[requestID]
	if !exists {
		return errors.New("request not found")
	}
	request.Status = types.RequestStatusCanceled
	return nil
}

// GetActiveSessions retrieves all sessions that are currently active.
func (s *InMemoryStorageAdapter) GetActiveSessions() ([]*types.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []*types.Session
	for _, session := range s.sessions {
		if session.Active {
			active = append(active, session)
		}
	}
	return active, nil
}
