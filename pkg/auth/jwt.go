package auth

import (
	"time"
	"github.com/dgrijalva/jwt-go"
)

// JWTSecret is the secret key for signing tokens (exported now)
var JWTSecret = []byte("your-secret-key")

// CustomClaims defines the structure of JWT claims
type CustomClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.StandardClaims
}

// GenerateJWT generates a JWT token for the user
func GenerateJWT(userID, role string) (string, error) {
	claims := CustomClaims{
		UserID: userID,
		Role:   role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret
	return token.SignedString(JWTSecret)
}

