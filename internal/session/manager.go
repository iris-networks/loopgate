package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"loopgate/internal/types"
)

type Manager struct {
	sessions     map[string]*types.Session
	chatSessions map[int64]string
	mu           sync.RWMutex
	dataFile     string
}

func NewManager(dataDir string) *Manager {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create data directory: %v", err))
	}

	m := &Manager{
		sessions:     make(map[string]*types.Session),
		chatSessions: make(map[int64]string),
		dataFile:     filepath.Join(dataDir, "sessions.json"),
	}

	m.loadSessions()
	return m
}

func (m *Manager) CreateSession(sessionID, clientID string, telegramID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[sessionID]; exists {
		return fmt.Errorf("session already exists: %s", sessionID)
	}

	session := &types.Session{
		ID:           sessionID,
		ClientID:     clientID,
		TelegramID:   telegramID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	m.sessions[sessionID] = session
	m.chatSessions[telegramID] = sessionID

	return m.saveSessions()
}

func (m *Manager) GetSession(sessionID string) (*types.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

func (m *Manager) GetSessionByChat(chatID int64) (*types.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessionID, exists := m.chatSessions[chatID]
	if !exists {
		return nil, fmt.Errorf("no session found for chat: %d", chatID)
	}

	return m.sessions[sessionID], nil
}

func (m *Manager) UpdateActivity(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.LastActivity = time.Now()
	return m.saveSessions()
}

func (m *Manager) DeactivateSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.IsActive = false
	delete(m.chatSessions, session.TelegramID)

	return m.saveSessions()
}

func (m *Manager) ListActiveSessions() []*types.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []*types.Session
	for _, session := range m.sessions {
		if session.IsActive {
			active = append(active, session)
		}
	}

	return active
}

func (m *Manager) CleanupInactiveSessions(maxAge time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	var toDelete []string

	for sessionID, session := range m.sessions {
		if !session.IsActive && session.LastActivity.Before(cutoff) {
			toDelete = append(toDelete, sessionID)
			delete(m.chatSessions, session.TelegramID)
		}
	}

	for _, sessionID := range toDelete {
		delete(m.sessions, sessionID)
	}

	if len(toDelete) > 0 {
		return m.saveSessions()
	}

	return nil
}

func (m *Manager) loadSessions() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var sessionData struct {
		Sessions     map[string]*types.Session `json:"sessions"`
		ChatSessions map[string]string         `json:"chat_sessions"`
	}

	if err := json.Unmarshal(data, &sessionData); err != nil {
		return err
	}

	m.sessions = sessionData.Sessions
	if m.sessions == nil {
		m.sessions = make(map[string]*types.Session)
	}

	m.chatSessions = make(map[int64]string)
	for chatIDStr, sessionID := range sessionData.ChatSessions {
		var chatID int64
		if _, err := fmt.Sscanf(chatIDStr, "%d", &chatID); err == nil {
			m.chatSessions[chatID] = sessionID
		}
	}

	return nil
}

func (m *Manager) saveSessions() error {
	chatSessionsStr := make(map[string]string)
	for chatID, sessionID := range m.chatSessions {
		chatSessionsStr[fmt.Sprintf("%d", chatID)] = sessionID
	}

	sessionData := struct {
		Sessions     map[string]*types.Session `json:"sessions"`
		ChatSessions map[string]string         `json:"chat_sessions"`
	}{
		Sessions:     m.sessions,
		ChatSessions: chatSessionsStr,
	}

	data, err := json.MarshalIndent(sessionData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.dataFile, data, 0644)
}

func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	active := 0
	inactive := 0

	for _, session := range m.sessions {
		if session.IsActive {
			active++
		} else {
			inactive++
		}
	}

	return map[string]interface{}{
		"total_sessions":    len(m.sessions),
		"active_sessions":   active,
		"inactive_sessions": inactive,
	}
}