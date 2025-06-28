package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"loopgate/internal/router"
	"loopgate/internal/session"
	"loopgate/internal/types"
)

type HITLHandler struct {
	router         *router.Router
	sessionManager *session.Manager
}

func NewHITLHandler(router *router.Router, sessionManager *session.Manager) *HITLHandler {
	return &HITLHandler{
		router:         router,
		sessionManager: sessionManager,
	}
}

func (h *HITLHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.HITLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if err := h.validateRequest(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	req.Timestamp = time.Now()
	if req.ID == "" {
		req.ID = fmt.Sprintf("hitl_%d", time.Now().UnixNano())
	}

	log.Printf("Processing HITL request: %s from client: %s", req.ID, req.ClientID)

	response, err := h.router.RouteHITLRequest(&req)
	if err != nil {
		log.Printf("Failed to route HITL request %s: %v", req.ID, err)
		http.Error(w, fmt.Sprintf("Failed to process request: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("HITL request %s completed with response: %s", req.ID, response.Response)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response for %s: %v", req.ID, err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *HITLHandler) HandleSessionRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var regReq struct {
		SessionID  string `json:"session_id"`
		ClientID   string `json:"client_id"`
		TelegramID int64  `json:"telegram_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&regReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if regReq.SessionID == "" || regReq.ClientID == "" || regReq.TelegramID == 0 {
		http.Error(w, "Missing required fields: session_id, client_id, telegram_id", http.StatusBadRequest)
		return
	}

	if err := h.sessionManager.CreateSession(regReq.SessionID, regReq.ClientID, regReq.TelegramID); err != nil {
		log.Printf("Failed to create session %s: %v", regReq.SessionID, err)
		http.Error(w, fmt.Sprintf("Failed to create session: %v", err), http.StatusConflict)
		return
	}

	log.Printf("Registered new session: %s for client: %s with Telegram ID: %d", 
		regReq.SessionID, regReq.ClientID, regReq.TelegramID)

	response := map[string]interface{}{
		"status":     "registered",
		"session_id": regReq.SessionID,
		"timestamp":  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HITLHandler) HandleSessionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id parameter is required", http.StatusBadRequest)
		return
	}

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Session not found: %v", err), http.StatusNotFound)
		return
	}

	pendingRequests := h.router.GetPendingRequestsForSession(sessionID)

	response := map[string]interface{}{
		"session":          session,
		"pending_requests": len(pendingRequests),
		"requests":         pendingRequests,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HITLHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.sessionManager.GetStats()
	stats["pending_requests"] = h.router.GetPendingRequestsCount()
	stats["timestamp"] = time.Now()
	stats["status"] = "healthy"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *HITLHandler) HandleSessionDeactivation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var deactReq struct {
		SessionID string `json:"session_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&deactReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if deactReq.SessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	if err := h.sessionManager.DeactivateSession(deactReq.SessionID); err != nil {
		log.Printf("Failed to deactivate session %s: %v", deactReq.SessionID, err)
		http.Error(w, fmt.Sprintf("Failed to deactivate session: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Deactivated session: %s", deactReq.SessionID)

	response := map[string]interface{}{
		"status":     "deactivated",
		"session_id": deactReq.SessionID,
		"timestamp":  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HITLHandler) validateRequest(req *types.HITLRequest) error {
	if req.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if req.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if req.Message == "" {
		return fmt.Errorf("message is required")
	}
	return nil
}