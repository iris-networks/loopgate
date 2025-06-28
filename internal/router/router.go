package router

import (
	"encoding/json"
	"io"
	"log"
	"loopgate/internal/handlers"
	"loopgate/internal/mcp"
	"net/http"

	"github.com/gorilla/mux"
)

type Router struct {
	mux         *mux.Router
	mcpServer   *mcp.Server
	hitlHandler *handlers.HITLHandler
}

func NewRouter(mcpServer *mcp.Server, hitlHandler *handlers.HITLHandler) *Router {
	router := &Router{
		mux:         mux.NewRouter(),
		mcpServer:   mcpServer,
		hitlHandler: hitlHandler,
	}

	router.setupRoutes()
	return router
}

func (r *Router) setupRoutes() {
	r.mux.HandleFunc("/health", r.healthCheck).Methods("GET")
	r.mux.HandleFunc("/mcp", r.handleMCP).Methods("POST")
	r.mux.HandleFunc("/mcp/tools", r.handleMCPTools).Methods("GET")
	r.mux.HandleFunc("/mcp/capabilities", r.handleMCPCapabilities).Methods("GET")

	r.hitlHandler.RegisterRoutes(r.mux)

	r.mux.Use(r.loggingMiddleware)
	r.mux.Use(r.corsMiddleware)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *Router) healthCheck(w http.ResponseWriter, req *http.Request) {
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "loopgate",
		"version": "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (r *Router) handleMCP(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	response, err := r.mcpServer.HandleHTTPRequest(body)
	if err != nil {
		http.Error(w, "Failed to process MCP request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func (r *Router) handleMCPTools(w http.ResponseWriter, req *http.Request) {
	response, err := r.mcpServer.CreateToolsListResponse()
	if err != nil {
		http.Error(w, "Failed to get tools list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func (r *Router) handleMCPCapabilities(w http.ResponseWriter, req *http.Request) {
	capabilities := r.mcpServer.GetCapabilities()
	serverInfo := r.mcpServer.GetServerInfo()

	response := map[string]interface{}{
		"capabilities": capabilities,
		"serverInfo":   serverInfo,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (r *Router) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Printf("%s %s %s", req.Method, req.RequestURI, req.RemoteAddr)
		next.ServeHTTP(w, req)
	})
}

func (r *Router) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if req.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, req)
	})
}