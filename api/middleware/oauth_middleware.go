package middleware

import (
	"caaspay-api-go/api/config"
	"context"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"net/http"
	"strings"
)

// OAuthMiddleware checks for a valid OAuth token in the Authorization header
func OAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
		RedirectURL:  cfg.OAuth.RedirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  cfg.OAuth.Endpoint.AuthURL,
			TokenURL: cfg.OAuth.Endpoint.TokenURL,
		},
	}

	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")

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
