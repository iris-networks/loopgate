package router

import (
	"encoding/json"
	"io"
	"log"
	"loopgate/config"
	"loopgate/internal/handlers"
	"loopgate/internal/mcp"
	"loopgate/internal/middleware"
	"loopgate/internal/storage"
	"net/http"

	"github.com/gorilla/mux"
)

type Router struct {
	mux            *mux.Router
	mcpServer      *mcp.Server
	hitlHandler    *handlers.HITLHandler
	authHandlers   *handlers.AuthHandlers
	userHandlers   *handlers.UserHandlers
	storageAdapter storage.StorageAdapter // Keep if needed for direct use, or pass to specific middleware/handlers
	cfg            *config.Config
}

func NewRouter(
	mcpServer *mcp.Server,
	hitlHandler *handlers.HITLHandler,
	storageAdapter storage.StorageAdapter,
	cfg *config.Config,
) *Router {
	authHandlers := handlers.NewAuthHandlers(storageAdapter, cfg.JWTSecretKey)
	userHandlers := handlers.NewUserHandlers(storageAdapter, cfg.APIKeyPrefix)

	router := &Router{
		mux:            mux.NewRouter(),
		mcpServer:      mcpServer,
		hitlHandler:    hitlHandler,
		authHandlers:   authHandlers,
		userHandlers:   userHandlers,
		storageAdapter: storageAdapter,
		cfg:            cfg,
	}

	router.setupRoutes()
	return router
}

func (r *Router) setupRoutes() {
	// Base middleware applied to all routes
	r.mux.Use(r.loggingMiddleware)
	r.mux.Use(r.corsMiddleware) // CORS should usually come before auth middlewares

	// Public routes
	r.mux.HandleFunc("/health", r.healthCheck).Methods("GET")

	// API Subrouter
	apiRouter := r.mux.PathPrefix("/api").Subrouter()

	// Auth routes (no JWT or API Key auth needed)
	authRouter := apiRouter.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/register", r.authHandlers.RegisterUserHandler).Methods("POST")
	authRouter.HandleFunc("/login", r.authHandlers.LoginUserHandler).Methods("POST")

	// User specific routes (protected by JWT)
	userRouter := apiRouter.PathPrefix("/user").Subrouter()
	userRouter.Use(middleware.JWTAuthMiddleware(r.cfg.JWTSecretKey))
	userRouter.HandleFunc("/apikeys", r.userHandlers.CreateAPIKeyHandler).Methods("POST")
	userRouter.HandleFunc("/apikeys", r.userHandlers.ListAPIKeysHandler).Methods("GET")
	userRouter.HandleFunc("/apikeys/{key_id}", r.userHandlers.RevokeAPIKeyHandler).Methods("DELETE")

	// Existing MCP and HITL routes
	// QUESTION for user: Should these be protected by APIKeyAuthMiddleware?
	// For now, leaving them as they were (public or protected by their own internal logic if any).
	// If they need protection:
	// mcpHitlProtectedRouter := r.mux.PathPrefix("").Subrouter() // Or specific prefix
	// mcpHitlProtectedRouter.Use(middleware.APIKeyAuthMiddleware(r.storageAdapter))
	// mcpHitlProtectedRouter.HandleFunc("/mcp", r.handleMCP).Methods("POST")
	// ... and so on for other routes

	r.mux.HandleFunc("/mcp", r.handleMCP).Methods("POST") // Example: Unprotected
	r.mux.HandleFunc("/mcp/tools", r.handleMCPTools).Methods("GET")
	r.mux.HandleFunc("/mcp/capabilities", r.handleMCPCapabilities).Methods("GET")
	if r.hitlHandler != nil { // hitlHandler might be nil if not configured/needed
		r.hitlHandler.RegisterRoutes(r.mux) // Assuming RegisterRoutes adds its own paths
	}

	// Example of a new route protected by API Key Authentication
	// saasProtectedRouter := apiRouter.PathPrefix("/saas").Subrouter()
	// saasProtectedRouter.Use(middleware.APIKeyAuthMiddleware(r.storageAdapter))
	// saasProtectedRouter.HandleFunc("/data", r.handleSaasData).Methods("GET") // handleSaasData would be a new handler
}

// Placeholder for a new SaaS-specific handler
// func (r *Router) handleSaasData(w http.ResponseWriter, req *http.Request) {
// 	 userID, ok := req.Context().Value(middleware.APIKeyUserContextKey).(uuid.UUID)
// 	 if !ok {
// 		 http.Error(w, "Could not identify user from API key", http.StatusInternalServerError)
// 		 return
// 	 }
//   // Process request using userID
// 	 w.Write([]byte("SaaS Data for user: " + userID.String()))
// }


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