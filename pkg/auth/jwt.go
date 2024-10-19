package auth

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"time"
)

// JWTSecret is the secret key for signing tokens (exported now)
var JWTSecret = []byte("your-secret-key")

// CustomClaims defines the structure of JWT claims
type CustomClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.StandardClaims
}

// GenerateJWT generates a JWT token for the user with a customizable expiration time
// expirationSconds is optional. If set to 0, it defaults to 1 hour.
func GenerateJWT(userID, role string, expirationSeconds ...int) (string, error) {
	// Set default expiration time to 1 hours if no value is passed
	expiration := 3600
	if len(expirationSeconds) > 0 && expirationSeconds[0] > 0 {
		expiration = expirationSeconds[0]
	}

	claims := CustomClaims{
		UserID: userID,
		Role:   role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(expiration) * time.Second).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret
	return token.SignedString(JWTSecret)
}

// ParseJWTToken parses and validates a JWT token string
func ParseJWTToken(tokenString string) (*CustomClaims, error) {
	// Parse the JWT and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	// Extract and return the claims
	if claims, ok := token.Claims.(*CustomClaims); ok {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

// RenewJWTToken renews a JWT token if it's within the renewal window (in seconds)
func RenewJWTToken(tokenString string, renewalWindowSeconds int) (string, error) {
	// Parse the existing token
	claims, err := ParseJWTToken(tokenString)
	if err != nil {
		return "", errors.New("invalid or expired token")
	}

	// Check if the token is within the renewal window (convert to time.Duration for comparison)
	if time.Until(time.Unix(claims.ExpiresAt, 0)) > time.Duration(renewalWindowSeconds)*time.Second {
		return "", errors.New("token is not within the renewal window")
	}

	// Generate a new token with the same user ID and role
	return GenerateJWT(claims.UserID, claims.Role)
}
