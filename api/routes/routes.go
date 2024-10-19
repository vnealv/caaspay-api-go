package routes

import (
	"caaspay-api-go/api/handlers"
	"caaspay-api-go/api/middleware"
	"caaspay-api-go/internal/rpc"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// ParamConfig defines the structure for route parameters
type ParamConfig struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Required    bool   `yaml:"required,omitempty"` // Defaults to false
	Description string `yaml:"description,omitempty"`
	Pattern     string `yaml:"pattern,omitempty"`
}

// RouteConfig represents the configuration for a single route
type RouteConfig struct {
	Path          string        `yaml:"path"`
	Type          string        `yaml:"type"`
	Authorization bool          `yaml:"authorization,omitempty"`
	AuthType      string        `yaml:"auth_type,omitempty"`
	Role          string        `yaml:"role,omitempty"`
	Service       string        `yaml:"service,omitempty"`
	Method        string        `yaml:"method,omitempty"`
	Params        []ParamConfig `yaml:"params"`
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
		// Validate and extract parameters
		args, err := validateAndExtractParams(c, routeConfig)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Determine the service and method
		service, method := getServiceAndMethod(c, routeConfig)

		// Get an RPC client from the pool
		rpcClient := rpcClientPool.GetClient()

		// Send the RPC request and get the response
		log.Printf("To call RPC: s:%v m:%v a:%v", service, method, args)
		response, err := rpcClient.CallRPC(service, method, args)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// Return the response to the client
		c.JSON(200, response)
	}
}

func validateAndExtractParams(c *gin.Context, routeConfig RouteConfig) (map[string]interface{}, error) {
	args := make(map[string]interface{})

	// Extract parameters from the request body for POST/PUT methods
	if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
		if err := c.ShouldBindJSON(&args); err != nil {
			return nil, fmt.Errorf("Error parsing/binding request body: %v", err.Error())
		}
	}

	log.Printf("BIND: %v", args)

	// Track allowed parameters and validate them
	allowedParams := map[string]bool{}
	for _, param := range routeConfig.Params {
		allowedParams[param.Name] = true
		// Check query and path params first
		queryValue, passedAsQuery := c.GetQuery(param.Name)
		pathValue := c.Param(param.Name)

		// Query parameters have higher precedence over body and path
		if passedAsQuery {
			args[param.Name] = queryValue
		} else if pathValue != "" {
			args[param.Name] = pathValue
		}

		// If the parameter is required but not provided
		if param.Required {
			if _, exists := args[param.Name]; !exists {
				return nil, fmt.Errorf("missing required parameter: %s - %s", param.Name, generateDescription(param))
			}
		}

		// If the parameter exists, validate its type and pattern
		if paramValue, exists := args[param.Name]; exists {
			value, ok := paramValue.(string)
			if !ok {
				return nil, fmt.Errorf("unable to parse parameter %s: - %s", param.Name, generateDescription(param))
			}
			switch param.Type {
			case "string":
				// Validate against the pattern if one is provided
				if param.Pattern != "" {
					matched, err := regexp.MatchString(param.Pattern, value)
					if err != nil || !matched {
						return nil, fmt.Errorf("invalid parameter value for %s: does not match pattern %s - %s", param.Name, param.Pattern, generateDescription(param))
					}
				}

			case "int":
				if _, err := strconv.Atoi(value); err != nil {
					return nil, fmt.Errorf("invalid parameter type for %s: expected int - %s", param.Name, generateDescription(param))
				}

			case "float":
				if _, err := strconv.ParseFloat(value, 64); err != nil {
					return nil, fmt.Errorf("invalid parameter type for %s: expected float - %s", param.Name, generateDescription(param))
				}

			case "bool":
				if _, err := strconv.ParseBool(value); err != nil {
					return nil, fmt.Errorf("invalid parameter type for %s: expected bool - %s", param.Name, generateDescription(param))
				}

			default:
				return nil, fmt.Errorf("unknown parameter type for %s", param.Name)
			}
		}
	}

	// Filter out extra parameters (not allowed by the config)
	for passedParamKey, _ := range args {
		if _, allowedKey := allowedParams[passedParamKey]; !allowedKey {
			log.Printf("Passed and extra param: %v", passedParamKey)
			delete(args, passedParamKey)
		}
	}

	return args, nil
}

// generateDescription auto-generates a description for a parameter if not provided
func generateDescription(param ParamConfig) string {
	description := fmt.Sprintf("%s (%s)", param.Name, param.Type)
	if param.Required {
		description += ", required"
	}
	if param.Pattern != "" {
		description += fmt.Sprintf(", pattern: %s", param.Pattern)
	}
	return description
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

	// Case 1: If the path is just `/`, set service to "api" and method to "request"
	if len(pathParts) == 1 && pathParts[0] == "" {
		return "api", "request"
	}

	// Case 2: If there is only one element in the path, use it as the service and default the method to "request"
	if len(pathParts) == 1 {
		return pathParts[0], "request"
	}

	// The last part of the path is the method, the rest is the service
	method := pathParts[len(pathParts)-1]
	service := strings.Join(pathParts[:len(pathParts)-1], ".")

	return service, method
}
