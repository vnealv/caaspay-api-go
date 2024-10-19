package routes

import (
	"caaspay-api-go/api/handlers"
	"caaspay-api-go/api/middleware"
	"caaspay-api-go/internal/rpc"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strings"
)

// RouteConfig represents the configuration for a single route
type RouteConfig struct {
	Path          string `yaml:"path"`
	Type          string `yaml:"type"`
	Authorization bool   `yaml:"authorization,omitempty"`
	AuthType      string `yaml:"auth_type,omitempty"`
	Role          string `yaml:"role,omitempty"`
	Service       string `yaml:"service,omitempty"`
	Method        string `yaml:"method,omitempty"`
}

// SetupRoutes loads the routes from the configuration and sets them up in Gin
func SetupRoutes(r *gin.Engine, rpcClientPool *rpc.RPCClientPool, configPath string) error {
	// Load the route configuration
	routeConfigs, err := LoadRouteConfigs(configPath)
	if err != nil {
		return fmt.Errorf("failed to load route configs: %w", err)
	}

	// Register the routes with middlewares
	for _, routeConfig := range routeConfigs {
		// Build the middleware stack
		mws := buildMiddlewareStack(r, routeConfig)

		// Register the route with the appropriate middlewares
		log.Printf("FF %v %v", routeConfig, mws)
		switch routeConfig.Type {
		case "GET":
			r.GET(routeConfig.Path, append(mws, createHandler(routeConfig, rpcClientPool))...)
		case "POST":
			r.POST(routeConfig.Path, append(mws, createHandler(routeConfig, rpcClientPool))...)
		default:
			fmt.Printf("Unsupported route type: %s for path: %s", routeConfig.Type, routeConfig.Path)
		}
	}

	return nil
}

// Global flags to ensure each login route is added only once
var jwtLoginRegistered = false
var oauthLoginRegistered = false

// buildMiddlewareStack creates the middleware stack for a given route
func buildMiddlewareStack(r *gin.Engine, route RouteConfig) []gin.HandlerFunc {
	mws := []gin.HandlerFunc{} // Middleware stack

	// Add authentication middleware based on auth_type
	if route.Authorization {
		switch route.AuthType {
		case "jwt":
			// Register the JWT login route if not already registered
			if !jwtLoginRegistered {
				r.POST("/jwt/login", handlers.JWTLoginHandler)
				r.POST("/jwt/renew", handlers.JWTRenewalHandler)
				jwtLoginRegistered = true
			}
			mws = append(mws, middleware.JWTAuthMiddleware())
		case "oauth":
			mws = append(mws, middleware.OAuthMiddleware())
		case "cloudflare_jwt":
			mws = append(mws, middleware.CloudflareJWTMiddleware())
		}
	}

	// Add RBAC middleware if a role is specified
	if route.Role != "" {
		mws = append(mws, middleware.RBACMiddleware(route.Role))
	}

	return mws
}

// createHandler dynamically creates a route handler based on the config and path
func createHandler(routeConfig RouteConfig, rpcClientPool *rpc.RPCClientPool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Determine the service and method
		service, method := getServiceAndMethod(c, routeConfig)

		// Get an RPC client from the pool
		rpcClient := rpcClientPool.GetClient()

		// Prepare the arguments by extracting from the request based on the method
		args := extractArgsFromRequest(c)

		// Send the RPC request and get the response
		response, err := rpcClient.CallRPC(service, method, args)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// Return the response to the client
		c.JSON(200, response)
	}
}

// extractArgsFromRequest extracts the request parameters (query, body, and path) and combines them into a single map
func extractArgsFromRequest(c *gin.Context) map[string]interface{} {
	args := make(map[string]interface{})

	// Extract query parameters (for GET, DELETE, etc.)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			args[key] = values[0]
		}
	}

	// Extract path parameters (from routes like /user/:id)
	for _, param := range c.Params {
		args[param.Key] = param.Value
	}

	// Extract body parameters (for POST, PUT, etc.) - assumes JSON body
	if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
		var bodyArgs map[string]interface{}
		if err := c.ShouldBindJSON(&bodyArgs); err == nil {
			for key, value := range bodyArgs {
				args[key] = value
			}
		}
	}

	return args
}

// getServiceAndMethod determines the service and method either from the config or the path
func getServiceAndMethod(c *gin.Context, routeConfig RouteConfig) (string, string) {
	// Use service and method from config if provided
	if routeConfig.Service != "" && routeConfig.Method != "" {
		service := strings.ReplaceAll(routeConfig.Service, "_", ".")
		method := routeConfig.Method
		return service, method
	}

	// Otherwise, derive from the request path
	path := strings.Trim(c.FullPath(), "/")
	pathParts := strings.Split(path, "/")

	if len(pathParts) < 2 {
		return "", ""
	}

	// The last part of the path is the method, the rest is the service
	method := pathParts[len(pathParts)-1]
	service := strings.Join(pathParts[:len(pathParts)-1], ".")

	return service, method
}
