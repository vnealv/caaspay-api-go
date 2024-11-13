package handlers

import (
	"caaspay-api-go/api/config"
	"caaspay-api-go/pkg/auth"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

// JWTLoginHandler authenticates a user and returns a JWT token
func JWTLoginHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var credentials struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := c.ShouldBindJSON(&credentials); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Verify allowed users and hash match
		var matchedUser *config.AllowedUser
		for _, user := range cfg.JWT.AllowedUsers {
			if user.Username == credentials.Username {
				if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err == nil {
					matchedUser = &user
					break
				}
			}
		}
		if matchedUser == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Generate JWT
		token, err := auth.GenerateJWT(cfg, matchedUser.Username, matchedUser.Role, int(cfg.JWT.TokenExpiry.Seconds()))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":   token,
			"expires": cfg.JWT.TokenExpiry.Seconds(),
		})
	}
}

// JWTRenewalHandler handles JWT token renewal
func JWTRenewalHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" || len(tokenString) < 7 || tokenString[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing or malformed"})
			return
		}
		tokenString = tokenString[7:]

		newToken, err := auth.RenewJWTToken(cfg, tokenString, int(cfg.JWT.TokenRenewalWindow.Seconds()))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": newToken})
	}
}
