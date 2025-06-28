package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"loopgate/internal/session"
	"loopgate/internal/telegram"
	"loopgate/internal/types"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type HITLHandler struct {
	sessionManager *session.Manager
	telegramBot    *telegram.Bot
}

func NewHITLHandler(sessionManager *session.Manager, telegramBot *telegram.Bot) *HITLHandler {
	return &HITLHandler{
		sessionManager: sessionManager,
		telegramBot:    telegramBot,
	}
}

func (h *HITLHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/hitl/register", h.RegisterSession).Methods("POST")
	router.HandleFunc("/hitl/request", h.SubmitRequest).Methods("POST")
	router.HandleFunc("/hitl/poll", h.PollRequest).Methods("GET")
	router.HandleFunc("/hitl/status", h.GetStatus).Methods("GET")
	router.HandleFunc("/hitl/deactivate", h.DeactivateSession).Methods("POST")
	router.HandleFunc("/hitl/pending", h.ListPendingRequests).Methods("GET")
	router.HandleFunc("/hitl/cancel", h.CancelRequest).Methods("POST")
}

func (h *HITLHandler) RegisterSession(w http.ResponseWriter, r *http.Request) {
	var req types.SessionRegistration
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" || req.ClientID == "" || req.TelegramID == 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	err := h.sessionManager.RegisterSession(req.SessionID, req.ClientID, req.TelegramID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to register session: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Registered session: %s for client: %s", req.SessionID, req.ClientID)

	response := map[string]interface{}{
		"success":    true,
		"session_id": req.SessionID,
		"message":    "Session registered successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HITLHandler) SubmitRequest(w http.ResponseWriter, r *http.Request) {
	var req types.HITLRequest
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" || req.ClientID == "" || req.Message == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	req.ID = uuid.New().String()
	req.Status = types.RequestStatusPending
	req.CreatedAt = time.Now()
	
	if req.Timeout == 0 {
		req.Timeout = 300
	}

	if req.RequestType == "" {
		if len(req.Options) > 0 {
			req.RequestType = types.RequestTypeChoice
		} else {
			req.RequestType = types.RequestTypeInput
		}
	}

	session, err := h.sessionManager.GetSession(req.SessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Session not found: %v", err), http.StatusNotFound)
		return
	}

	if !session.Active {
		http.Error(w, "Session is not active", http.StatusBadRequest)
		return
	}

	h.sessionManager.StoreRequest(&req)

	err = h.telegramBot.SendHITLRequest(&req)
	if err != nil {
		log.Printf("Failed to send telegram message: %v", err)
		http.Error(w, "Failed to send request to Telegram", http.StatusInternalServerError)
		return
	}

	log.Printf("Submitted HITL request: %s for client: %s", req.ID, req.ClientID)

	response := map[string]interface{}{
		"success":    true,
		"request_id": req.ID,
		"status":     req.Status,
		"created_at": req.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HITLHandler) PollRequest(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		http.Error(w, "Missing request_id parameter", http.StatusBadRequest)
		return
	}

	request, err := h.sessionManager.GetRequest(requestID)
	if err != nil {
		http.Error(w, "Request not found", http.StatusNotFound)
		return
	}

	response := types.PollResponse{
		RequestID: requestID,
		Status:    request.Status,
		Response:  request.Response,
		Approved:  request.Approved,
		Completed: request.Status == types.RequestStatusCompleted ||
		          request.Status == types.RequestStatusTimeout ||
		          request.Status == types.RequestStatusCanceled,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HITLHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "Missing session_id parameter", http.StatusBadRequest)
		return
	}

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (h *HITLHandler) DeactivateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		http.Error(w, "Missing session_id", http.StatusBadRequest)
		return
	}

	err := h.sessionManager.DeactivateSession(req.SessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to deactivate session: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Deactivated session: %s", req.SessionID)

	response := map[string]interface{}{
		"success": true,
		"message": "Session deactivated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HITLHandler) ListPendingRequests(w http.ResponseWriter, r *http.Request) {
	pending := h.sessionManager.GetPendingRequests()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pending_requests": pending,
		"count":           len(pending),
	})
}

func (h *HITLHandler) CancelRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequestID string `json:"request_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.RequestID == "" {
		http.Error(w, "Missing request_id", http.StatusBadRequest)
		return
	}

	err := h.sessionManager.CancelRequest(req.RequestID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to cancel request: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Canceled request: %s", req.RequestID)

	response := map[string]interface{}{
		"success": true,
		"message": "Request canceled successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}