package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"loopgate/internal/types"
)

type Bot struct {
	token       string
	sessions    map[string]*types.Session
	chatSessions map[int64]string
	mu          sync.RWMutex
	mcpServer   MCPHandler
}

type MCPHandler interface {
	HandleTelegramResponse(requestID string, response *types.HITLResponse) error
}

type TelegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		MessageID int `json:"message_id"`
		Chat      struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
	CallbackQuery *struct {
		ID   string `json:"id"`
		Data string `json:"data"`
		Message struct {
			MessageID int `json:"message_id"`
			Chat      struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
	} `json:"callback_query"`
}

func NewBot(token string) *Bot {
	return &Bot{
		token:        token,
		sessions:     make(map[string]*types.Session),
		chatSessions: make(map[int64]string),
	}
}

func (b *Bot) SetMCPHandler(handler MCPHandler) {
	b.mcpServer = handler
}

func (b *Bot) SendMessage(chatID int64, message string, options []string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.token)
	
	var keyboard interface{}
	if len(options) > 0 {
		buttons := make([][]map[string]string, len(options))
		for i, option := range options {
			buttons[i] = []map[string]string{
				{
					"text":          option,
					"callback_data": fmt.Sprintf("option_%d", i),
				},
			}
		}
		keyboard = map[string]interface{}{
			"inline_keyboard": buttons,
		}
	}

	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    message,
	}
	
	if keyboard != nil {
		payload["reply_markup"] = keyboard
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API error: %d", resp.StatusCode)
	}

	return nil
}

func (b *Bot) RegisterSession(sessionID, clientID string, chatID int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	session := &types.Session{
		ID:           sessionID,
		ClientID:     clientID,
		TelegramID:   chatID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	b.sessions[sessionID] = session
	b.chatSessions[chatID] = sessionID

	return nil
}

func (b *Bot) GetChatID(sessionID string) (int64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	session, exists := b.sessions[sessionID]
	if !exists {
		return 0, fmt.Errorf("session not found: %s", sessionID)
	}

	return session.TelegramID, nil
}

func (b *Bot) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var update TelegramUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if update.Message != nil && update.Message.Text != "" {
		b.handleTextMessage(update.Message.Chat.ID, update.Message.Text)
	} else if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)
	}

	w.WriteHeader(http.StatusOK)
}

func (b *Bot) handleTextMessage(chatID int64, text string) {
	b.mu.RLock()
	sessionID, exists := b.chatSessions[chatID]
	b.mu.RUnlock()

	if !exists {
		b.SendMessage(chatID, "Session not registered. Please register your session first.", nil)
		return
	}

	response := &types.HITLResponse{
		ID:       "unknown",
		Response: text,
		Approved: strings.ToLower(text) == "yes" || strings.ToLower(text) == "approve",
		Time:     time.Now(),
	}

	if b.mcpServer != nil {
		b.mcpServer.HandleTelegramResponse(sessionID, response)
	}
}

func (b *Bot) handleCallbackQuery(query *struct {
	ID   string `json:"id"`
	Data string `json:"data"`
	Message struct {
		MessageID int `json:"message_id"`
		Chat      struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
}) {
	chatID := query.Message.Chat.ID

	b.mu.RLock()
	sessionID, exists := b.chatSessions[chatID]
	b.mu.RUnlock()

	if !exists {
		return
	}

	selected := -1
	if strings.HasPrefix(query.Data, "option_") {
		if idx, err := strconv.Atoi(strings.TrimPrefix(query.Data, "option_")); err == nil {
			selected = idx
		}
	}

	response := &types.HITLResponse{
		ID:       generateResponseID(),
		Response: query.Data,
		Selected: selected,
		Approved: true,
		Time:     time.Now(),
	}

	if b.mcpServer != nil {
		b.mcpServer.HandleTelegramResponse(sessionID, response)
	}

	b.answerCallbackQuery(query.ID)
}

func (b *Bot) answerCallbackQuery(queryID string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", b.token)
	payload := map[string]string{"callback_query_id": queryID}
	
	jsonPayload, _ := json.Marshal(payload)
	http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
}

func (b *Bot) StartPolling() {
	offset := 0
	for {
		updates, err := b.getUpdates(offset)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1
			go b.processUpdate(update)
		}

		if len(updates) == 0 {
			time.Sleep(time.Second)
		}
	}
}

func (b *Bot) getUpdates(offset int) ([]TelegramUpdate, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d", b.token, offset)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool              `json:"ok"`
		Result []TelegramUpdate `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram API error")
	}

	return result.Result, nil
}

func (b *Bot) processUpdate(update TelegramUpdate) {
	if update.Message != nil && update.Message.Text != "" {
		b.handleTextMessage(update.Message.Chat.ID, update.Message.Text)
	} else if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)
	}
}

func generateResponseID() string {
	return fmt.Sprintf("resp_%d", time.Now().UnixNano())
}