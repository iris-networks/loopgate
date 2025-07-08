package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const (
	// DefaultAPIKeyPrefix is the default prefix for generated API keys.
	// Example: lk_pub_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
	DefaultAPIKeyPrefix = "lk_pub_"
	apiKeyLengthBytes = 32 // Generates a 64-character hex string
)

// GenerateAPIKey creates a new API key, returning the full key (for user display once)
// and its SHA-256 hash (for storage).
// The prefix helps identify the key type but is part of the key that gets hashed.
func GenerateAPIKey(prefix string) (rawKey string, keyHash string, err error) {
	if prefix == "" {
		prefix = DefaultAPIKeyPrefix
	}

	randomBytes := make([]byte, apiKeyLengthBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes for API key: %w", err)
	}

	keySuffix := hex.EncodeToString(randomBytes)
	rawKey = prefix + keySuffix

	// Hash the raw key for storage
	hash := sha256.Sum256([]byte(rawKey))
	keyHash = hex.EncodeToString(hash[:])

	return rawKey, keyHash, nil
}
