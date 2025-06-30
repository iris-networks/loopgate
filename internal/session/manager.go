package session

import (
	// "errors" // Removed unused import
	"loopgate/internal/storage"
	"loopgate/internal/types"
	// "time" // Removed unused import
)

type Manager struct {
	adapter storage.StorageAdapter
}

func NewManager(adapter storage.StorageAdapter) *Manager {
	return &Manager{
		adapter: adapter,
	}
}

func (m *Manager) RegisterSession(sessionID, clientID string, telegramID int64) error {
	return m.adapter.RegisterSession(sessionID, clientID, telegramID)
}

func (m *Manager) DeactivateSession(sessionID string) error {
	return m.adapter.DeactivateSession(sessionID)
}

func (m *Manager) GetSession(sessionID string) (*types.Session, error) {
	return m.adapter.GetSession(sessionID)
}

func (m *Manager) GetTelegramID(clientID string) (int64, error) {
	return m.adapter.GetTelegramID(clientID)
}

func (m *Manager) StoreRequest(request *types.HITLRequest) error {
	return m.adapter.StoreRequest(request)
}

func (m *Manager) GetRequest(requestID string) (*types.HITLRequest, error) {
	return m.adapter.GetRequest(requestID)
}

func (m *Manager) UpdateRequestResponse(requestID, response string, approved bool) error {
	return m.adapter.UpdateRequestResponse(requestID, response, approved)
}

func (m *Manager) GetPendingRequests() ([]*types.HITLRequest, error) {
	return m.adapter.GetPendingRequests()
}

func (m *Manager) CancelRequest(requestID string) error {
	return m.adapter.CancelRequest(requestID)
}

func (m *Manager) GetActiveSessions() ([]*types.Session, error) {
	return m.adapter.GetActiveSessions()
}