package router

import (
	"fmt"
	"log"
	"sync"
	"time"

	"loopgate/internal/session"
	"loopgate/internal/telegram"
	"loopgate/internal/types"
)

type Router struct {
	sessionManager *session.Manager
	telegramBot    *telegram.Bot
	pendingRequests map[string]*PendingRequest
	mu             sync.RWMutex
}

type PendingRequest struct {
	Request     *types.HITLRequest
	ResponseCh  chan *types.HITLResponse
	Timeout     time.Time
	SessionInfo *types.Session
}

func NewRouter(sessionManager *session.Manager, telegramBot *telegram.Bot) *Router {
	r := &Router{
		sessionManager:  sessionManager,
		telegramBot:     telegramBot,
		pendingRequests: make(map[string]*PendingRequest),
	}

	go r.cleanupExpiredRequests()
	return r
}

func (r *Router) RouteHITLRequest(req *types.HITLRequest) (*types.HITLResponse, error) {
	session, err := r.sessionManager.GetSession(req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %v", err)
	}

	if !session.IsActive {
		return nil, fmt.Errorf("session is not active: %s", req.SessionID)
	}

	responseCh := make(chan *types.HITLResponse, 1)
	timeout := time.Now().Add(5 * time.Minute)

	pendingReq := &PendingRequest{
		Request:     req,
		ResponseCh:  responseCh,
		Timeout:     timeout,
		SessionInfo: session,
	}

	r.mu.Lock()
	r.pendingRequests[req.ID] = pendingReq
	r.mu.Unlock()

	if err := r.sessionManager.UpdateActivity(req.SessionID); err != nil {
		log.Printf("Failed to update session activity: %v", err)
	}

	message := r.formatMessage(req)
	if err := r.telegramBot.SendMessage(session.TelegramID, message, req.Options); err != nil {
		r.mu.Lock()
		delete(r.pendingRequests, req.ID)
		r.mu.Unlock()
		return nil, fmt.Errorf("failed to send telegram message: %v", err)
	}

	select {
	case response := <-responseCh:
		r.mu.Lock()
		delete(r.pendingRequests, req.ID)
		r.mu.Unlock()
		return response, nil
	case <-time.After(time.Until(timeout)):
		r.mu.Lock()
		delete(r.pendingRequests, req.ID)
		r.mu.Unlock()
		return nil, fmt.Errorf("request timeout for ID: %s", req.ID)
	}
}

func (r *Router) HandleTelegramResponse(sessionID string, response *types.HITLResponse) error {
	r.mu.RLock()
	pendingReq, exists := r.pendingRequests[response.ID]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no pending request found for ID: %s", response.ID)
	}

	select {
	case pendingReq.ResponseCh <- response:
		return nil
	default:
		return fmt.Errorf("response channel full for request: %s", response.ID)
	}
}

func (r *Router) HandleTelegramMessage(chatID int64, message string, callbackData string) error {
	session, err := r.sessionManager.GetSessionByChat(chatID)
	if err != nil {
		return fmt.Errorf("session not found for chat %d: %v", chatID, err)
	}

	r.mu.RLock()
	var pendingReq *PendingRequest
	for _, req := range r.pendingRequests {
		if req.SessionInfo.ID == session.ID {
			pendingReq = req
			break
		}
	}
	r.mu.RUnlock()

	if pendingReq == nil {
		return fmt.Errorf("no pending request found for session: %s", session.ID)
	}

	response := &types.HITLResponse{
		ID:       pendingReq.Request.ID,
		Response: message,
		Time:     time.Now(),
	}

	if callbackData != "" {
		response.Response = callbackData
		if callbackData == "approve" {
			response.Approved = true
		} else if callbackData == "reject" {
			response.Approved = false
		} else {
			response.Approved = true
		}
	} else {
		response.Approved = r.parseApproval(message)
	}

	select {
	case pendingReq.ResponseCh <- response:
		return nil
	default:
		return fmt.Errorf("response channel full for request: %s", pendingReq.Request.ID)
	}
}

func (r *Router) formatMessage(req *types.HITLRequest) string {
	message := fmt.Sprintf("🤖 *HITL Request from %s*\n\n", req.ClientID)
	message += fmt.Sprintf("📝 *Message:* %s\n", req.Message)
	
	if req.Metadata != nil && len(req.Metadata) > 0 {
		message += "\n📊 *Metadata:*\n"
		for key, value := range req.Metadata {
			message += fmt.Sprintf("• %s: %v\n", key, value)
		}
	}

	if len(req.Options) == 0 {
		message += "\n💬 Please respond with your decision (yes/no, approve/reject)."
	} else {
		message += "\n🔘 Please select one of the options below:"
	}

	message += fmt.Sprintf("\n\n⏰ *Request ID:* `%s`", req.ID)
	return message
}

func (r *Router) parseApproval(message string) bool {
	lower := fmt.Sprintf("%s", message)
	approvals := []string{"yes", "approve", "ok", "confirm", "accept", "✅", "👍"}
	
	for _, approval := range approvals {
		if lower == approval {
			return true
		}
	}
	
	return false
}

func (r *Router) cleanupExpiredRequests() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		var expired []string

		r.mu.RLock()
		for id, req := range r.pendingRequests {
			if now.After(req.Timeout) {
				expired = append(expired, id)
			}
		}
		r.mu.RUnlock()

		if len(expired) > 0 {
			r.mu.Lock()
			for _, id := range expired {
				if req, exists := r.pendingRequests[id]; exists {
					close(req.ResponseCh)
					delete(r.pendingRequests, id)
					log.Printf("Expired HITL request: %s", id)
				}
			}
			r.mu.Unlock()
		}
	}
}

func (r *Router) GetPendingRequestsCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.pendingRequests)
}

func (r *Router) GetPendingRequestsForSession(sessionID string) []*types.HITLRequest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var requests []*types.HITLRequest
	for _, req := range r.pendingRequests {
		if req.SessionInfo.ID == sessionID {
			requests = append(requests, req.Request)
		}
	}

	return requests
}