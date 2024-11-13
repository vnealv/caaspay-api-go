package auth

import (
	"caaspay-api-go/api/config"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"time"
)

// CustomClaims defines the structure of JWT claims
type CustomClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a new JWT token using settings from cfg
func GenerateJWT(cfg *config.Config, userID, role string, expirationSeconds int) (string, error) {
	claims := CustomClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expirationSeconds) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.JWTSecret))
}

// RenewJWTToken renews an existing JWT token if it is within the renewal window.
func RenewJWTToken(cfg *config.Config, tokenString string, renewalWindowSeconds int) (string, error) {
	claims, err := ParseJWTToken(cfg, tokenString)
	if err != nil {
		return "", errors.New("invalid or expired token")
	}

	if time.Until(claims.ExpiresAt.Time) > time.Duration(renewalWindowSeconds)*time.Second {
		return "", errors.New("token is not within the renewal window")
	}

	return GenerateJWT(cfg, claims.UserID, claims.Role, int(cfg.JWT.TokenExpiry.Seconds()))
}

// ParseJWTToken parses and validates a JWT token string using the secret from cfg
func ParseJWTToken(cfg *config.Config, tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWT.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	claims, ok := token.Claims.(*CustomClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}
