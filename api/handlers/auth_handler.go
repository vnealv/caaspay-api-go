package handlers

import (
	"caaspay-api-go/pkg/auth"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

// Mocked user data with hashed passwords
// Replace this with a secure database lookup in production
var users = map[string]string{
	"admin": hashPassword("password123"), // bcrypt-hashed password
	"user":  hashPassword("mypassword"),
}

var tokenExpiry = int((15 * time.Minute).Seconds())

// hashPassword hashes the password using bcrypt
func hashPassword(password string) string {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err) // In production, handle this more gracefully
	}
	return string(hashedPassword)
}

// JWTLoginHandler authenticates a user and returns a JWT token
func JWTLoginHandler(c *gin.Context) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	// Parse the incoming login request
	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Check if the user exists
	storedHashedPassword, ok := users[credentials.Username]
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Compare the hashed password with the password provided by the user
	if err := bcrypt.CompareHashAndPassword([]byte(storedHashedPassword), []byte(credentials.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Assign role based on username (for testing purposes)
	role := "user"
	if credentials.Username == "admin" {
		role = "admin"
	}

	// Generate JWT token
	token, err := auth.GenerateJWT(credentials.Username, role, tokenExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	// Send the JWT token as a response
	c.JSON(http.StatusOK, gin.H{
		"token":   token,
		"expires": tokenExpiry,
	})
}

// JWTRenewalHandler handles JWT token renewal
func JWTRenewalHandler(c *gin.Context) {
	// Get the existing token from the Authorization header
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" || len(tokenString) < 7 || tokenString[:7] != "Bearer " {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing or malformed"})
		return
	}
	tokenString = tokenString[7:] // Remove "Bearer " prefix

	// Renew the token if it's within the 15-minute renewal window
	newTokenString, err := auth.RenewJWTToken(tokenString, tokenExpiry)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Return the new token in the response
	c.JSON(http.StatusOK, gin.H{"token": newTokenString})
}
