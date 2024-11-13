package middleware

import (
	"caaspay-api-go/api/config"
	"caaspay-api-go/pkg/auth"
	"context"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"net/http"
	"strings"
)

// JWTAuthMiddleware checks for valid JWT token in Authorization header
func JWTAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")

		// Check if Authorization header is present and formatted correctly
		if tokenString == "" || !strings.HasPrefix(tokenString, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing or malformed"})
			c.Abort()
			return
		}

		tokenString = tokenString[7:] // Remove "Bearer " prefix

		// Parse the JWT and validate the token
		token, err := jwt.ParseWithClaims(tokenString, &auth.CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return auth.JWTSecret, nil // Fix here: Export the jwtSecret from auth package
		})

		// Handle JWT parsing errors
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Extract the claims and pass the user information to the context
		if claims, ok := token.Claims.(*auth.CustomClaims); ok {
			c.Set("userID", claims.UserID)
			c.Set("role", claims.Role)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OAuth configuration for your service (replace with your own settings)
var oauthConfig = oauth2.Config{
	ClientID:     "your-client-id",
	ClientSecret: "your-client-secret",
	RedirectURL:  "your-redirect-url",
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://provider.com/oauth/authorize",
		TokenURL: "https://provider.com/oauth/token",
	},
}

// OAuthMiddleware checks for a valid OAuth token in the Authorization header
func OAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")

		// Check if Authorization header is present and formatted correctly
		if tokenString == "" || !strings.HasPrefix(tokenString, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing or malformed"})
			c.Abort()
			return
		}

		token := tokenString[7:] // Remove "Bearer " prefix

		// Validate the OAuth token using the token source
		tokenSource := oauthConfig.TokenSource(context.Background(), &oauth2.Token{
			AccessToken: token,
		})

		_, err := tokenSource.Token()
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid OAuth token"})
			c.Abort()
			return
		}

		// If the token is valid, continue processing
		c.Next()
	}
}
