package auth

import (
	"fmt"
	"loopgate/internal/types"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	defaultTokenDuration = 24 * time.Hour
)

// GenerateJWT creates a new JWT for a given user.
func GenerateJWT(userID uuid.UUID, username string, jwtSecret string) (string, error) {
	if jwtSecret == "" {
		return "", fmt.Errorf("JWT secret cannot be empty")
	}

	expirationTime := time.Now().Add(defaultTokenDuration)
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   userID.String(), // Using Subject field for UserID
		Issuer:    "loopgate",      // Optional: identify the issuer
		// Custom claims will be part of our types.Claims struct wrapper if needed,
		// but for this, we'll make a structure that embeds RegisteredClaims
	}

	// Create a new struct that embeds jwt.RegisteredClaims and adds our custom fields
	customClaims := &struct {
		jwt.RegisteredClaims
		UserID   uuid.UUID `json:"user_id"`
		Username string    `json:"username"`
	}{
		RegisteredClaims: *claims,
		UserID:           userID,
		Username:         username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, customClaims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateJWT validates a JWT string and returns the custom claims if valid.
func ValidateJWT(tokenString string, jwtSecret string) (*types.Claims, error) {
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT secret cannot be empty")
	}

	// Define the structure for custom claims to parse into.
	// This must match the structure used during token generation.
	parsedClaims := &struct {
		jwt.RegisteredClaims
		UserID   uuid.UUID `json:"user_id"`
		Username string    `json:"username"`
	}{}

	token, err := jwt.ParseWithClaims(tokenString, parsedClaims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	// Map the parsed claims to our internal types.Claims struct
	appClaims := &types.Claims{
		UserID:         parsedClaims.UserID,
		Username:       parsedClaims.Username,
		RegisteredClaims: parsedClaims.RegisteredClaims, // Store the standard claims
	}

	return appClaims, nil
}
