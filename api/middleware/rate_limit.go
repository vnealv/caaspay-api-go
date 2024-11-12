package middleware

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"net/http"
	"sync"
)

// rateLimiterStore holds the rate limiters for each route path.
var rateLimiterStore = sync.Map{}

// getOrCreateRateLimiter creates or retrieves a rate limiter for a given path.
func getOrCreateRateLimiter(path string, limit rate.Limit, burst int) *rate.Limiter {
	if val, exists := rateLimiterStore.Load(path); exists {
		return val.(*rate.Limiter)
	}
	limiter := rate.NewLimiter(limit, burst)
	rateLimiterStore.Store(path, limiter)
	return limiter
}

// RateLimitMiddleware returns a Gin middleware that applies rate limiting to each route based on its path.
func RateLimitMiddleware(path string, requestsPerSecond int, burst int) gin.HandlerFunc {
	limiter := getOrCreateRateLimiter(path, rate.Limit(requestsPerSecond), burst)
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
