package session

import (
	"errors"
	"loopgate/internal/types"
	"sync"
	"time"
)

type Manager struct {
	sessions map[string]*types.Session
	requests map[string]*types.HITLRequest
	clientToTelegram map[string]int64
	mu       sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		sessions:         make(map[string]*types.Session),
		requests:         make(map[string]*types.HITLRequest),
		clientToTelegram: make(map[string]int64),
	}
}

func (m *Manager) RegisterSession(sessionID, clientID string, telegramID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := &types.Session{
		ID:         sessionID,
		ClientID:   clientID,
		TelegramID: telegramID,
		Active:     true,
		CreatedAt:  time.Now(),
	}

	m.sessions[sessionID] = session
	m.clientToTelegram[clientID] = telegramID

	return nil
}

func (m *Manager) DeactivateSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return errors.New("session not found")
	}

	session.Active = false
	delete(m.clientToTelegram, session.ClientID)

	return nil
}

func (m *Manager) GetSession(sessionID string) (*types.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, errors.New("session not found")
	}

	return session, nil
}

func (m *Manager) GetTelegramID(clientID string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	telegramID, exists := m.clientToTelegram[clientID]
	if !exists {
		return 0, errors.New("client not found")
	}

	return telegramID, nil
}

func (m *Manager) StoreRequest(request *types.HITLRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests[request.ID] = request
}

func (m *Manager) GetRequest(requestID string) (*types.HITLRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	request, exists := m.requests[requestID]
	if !exists {
		return nil, errors.New("request not found")
	}

	return request, nil
}

func (m *Manager) UpdateRequestResponse(requestID, response string, approved bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	request, exists := m.requests[requestID]
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

func (m *Manager) GetPendingRequests() []*types.HITLRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var pending []*types.HITLRequest
	for _, request := range m.requests {
		if request.Status == types.RequestStatusPending {
			pending = append(pending, request)
		}
	}

	return pending
}

func (m *Manager) CancelRequest(requestID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	request, exists := m.requests[requestID]
	if !exists {
		return errors.New("request not found")
	}

	request.Status = types.RequestStatusCanceled
	return nil
}

func (m *Manager) GetActiveSessions() []*types.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []*types.Session
	for _, session := range m.sessions {
		if session.Active {
			active = append(active, session)
		}
	}

	return active
}