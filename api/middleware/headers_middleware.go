package middleware

import (
	"caaspay-api-go/internal/logging"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"net/http"
)

// SecurityHeadersMiddleware applies secure headers for production.
func SecurityHeadersMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers for XSS, clickjacking, and content type
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")

		// Content-Security-Policy restricts external resources
		// will prevent swagger from loading if enabled
		//c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; object-src 'none'")

		// Strict-Transport-Security enforces HTTPS
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Referrer-Policy minimizes data shared in referrer headers
		c.Writer.Header().Set("Referrer-Policy", "no-referrer")

		// Permissions-Policy restricts use of features like camera and microphone
		c.Writer.Header().Set("Permissions-Policy", "fullscreen=(self)")

		// Cross-Origin Resource Sharing (CORS) headers
		origin := c.Request.Header.Get("Origin")
		for _, o := range allowedOrigins {
			if o == origin {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight OPTIONS requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

var cloudflareHeaders = []string{
	"CF-Connecting-IP",  // Original visitor IP
	"CF-IPCountry",      // Country code
	"CF-Ray",            // Unique request ID in Cloudflare
	"CF-Visitor",        // JSON with scheme info
	"X-Forwarded-For",   // Proxied IP
	"X-Forwarded-Proto", // Original protocol
}

// CloudflareMiddleware adds Cloudflare headers to logs and OpenTelemetry trace.
func CloudflareMiddleware(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set up OpenTelemetry span
		tracer := otel.Tracer("caaspay-api")
		ctx, span := tracer.Start(c.Request.Context(), "HTTP Request")
		defer span.End()

		// Capture Cloudflare headers if present
		cfHeaders := make(map[string]string)
		for _, header := range cloudflareHeaders {
			if value := c.GetHeader(header); value != "" {
				cfHeaders[header] = value
				c.Writer.Header().Set(header, value) // Include in response headers for debugging if needed
			}
		}

		// Log request with Cloudflare headers
		logger.LogWithStats("info", "Incoming request",
			map[string]string{"path": c.Request.URL.Path, "method": c.Request.Method},
			map[string]interface{}{"cloudflare_headers": cfHeaders},
		)

		// Add Cloudflare headers as attributes in OpenTelemetry
		for key, value := range cfHeaders {
			span.SetAttributes(attribute.String(key, value))
		}

		// to add IP Whitelisting here
		// depending on CF-Connecting-IP

		// Pass the updated context into the request
		c.Request = c.Request.WithContext(ctx)

		c.Next() // Continue to the next handler

		// Log the status after response
		status := c.Writer.Status()
		logger.LogWithStats("info", "Request completed",
			map[string]string{"status": http.StatusText(status), "status_code": fmt.Sprintf("%d", status)},
			map[string]interface{}{"cloudflare_headers": cfHeaders},
		)
	}
}
