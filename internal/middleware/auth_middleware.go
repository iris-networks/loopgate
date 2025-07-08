package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"loopgate/internal/auth"
	"loopgate/internal/storage"
)

type contextKey string

const UserClaimsContextKey = contextKey("userClaims")
const APIKeyUserContextKey = contextKey("apiKeyUser") // To store UserID of the API key owner

// JWTAuthMiddleware protects routes that require a logged-in user via JWT.
// It extracts user claims from the JWT and adds them to the request context.
func JWTAuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "Authorization header format must be Bearer {token}", http.StatusUnauthorized)
				return
			}
			tokenString := parts[1]

			claims, err := auth.ValidateJWT(tokenString, jwtSecret)
			if err != nil {
				http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Add claims to context
			ctx := context.WithValue(r.Context(), UserClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// APIKeyAuthMiddleware protects routes that require API key authentication.
// It validates the API key and can add authenticated user info to the context.
func APIKeyAuthMiddleware(storageAdapter storage.StorageAdapter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKeyHeader := r.Header.Get("Authorization")
			if apiKeyHeader == "" {
				// Fallback: check X-API-Key header as well, common practice
				apiKeyHeader = r.Header.Get("X-API-Key")
				if apiKeyHeader == "" {
					http.Error(w, "API key required (Authorization: Bearer <key> or X-API-Key: <key>)", http.StatusUnauthorized)
					return
				}
				// If X-API-Key is used, it's directly the key
			} else {
				// If Authorization header is used, expect "Bearer <key>"
				parts := strings.Split(apiKeyHeader, " ")
				if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
					http.Error(w, "API key format must be Bearer {key} if using Authorization header", http.StatusUnauthorized)
					return
				}
				apiKeyHeader = parts[1]
			}


			if apiKeyHeader == "" { // Should be caught by above, but as a safeguard
				http.Error(w, "API key cannot be empty", http.StatusUnauthorized)
				return
			}

			// Hash the provided raw key to compare with stored hash
			hash := sha256.Sum256([]byte(apiKeyHeader))
			keyHash := hex.EncodeToString(hash[:])

			apiKey, err := storageAdapter.GetActiveAPIKeyByHash(keyHash)
			if err != nil {
				// Log the actual error for server-side debugging if needed, but return generic error to client
				// log.Printf("API key validation error: %v (for hash: %s)", err, keyHash)
				http.Error(w, "Invalid or inactive API key", http.StatusUnauthorized)
				return
			}

			// Key is valid and active, update LastUsedAt (best effort, don't fail request if this errors)
			_ = storageAdapter.UpdateAPIKeyLastUsed(apiKey.ID)

			// Add API key owner's UserID to context for downstream handlers
			// This allows handlers to know which user is making the API call via this key.
			ctxWithUser := context.WithValue(r.Context(), APIKeyUserContextKey, apiKey.UserID)

			// Optionally, also add the APIKey ID itself to context if needed
			// ctxWithAPIKey := context.WithValue(ctxWithUser, "apiKeyID", apiKey.ID)

			next.ServeHTTP(w, r.WithContext(ctxWithUser))
		})
	}
}
