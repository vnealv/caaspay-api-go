package middleware

import (
	"caaspay-api-go/api/config"
	"caaspay-api-go/internal/auth"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

// JWTAuthMiddleware checks for a valid JWT token in Authorization header.
func JWTAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")

		// Validate token presence and format
		if tokenString == "" || !strings.HasPrefix(tokenString, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing or malformed"})
			c.Abort()
			return
		}
		tokenString = tokenString[7:] // Remove "Bearer " prefix

		// Parse and validate JWT token
		claims, err := auth.ParseJWTToken(cfg, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Store user information in context
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)

		c.Next() // Continue to the next handler
	}
}
