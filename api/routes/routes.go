package routes

import (
	"caaspay-api-go/api/config"
	"caaspay-api-go/api/handlers"
	"caaspay-api-go/api/middleware"
	"caaspay-api-go/internal/logging"
	"caaspay-api-go/internal/rpc"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// RouteConfig represents the configuration for a single route
type RouteConfig struct {
	Path              string               `mapstructure:"path"`
	Type              string               `mapstructure:"type"`
	Authorization     bool                 `mapstructure:"authorization"`
	AuthType          string               `mapstructure:"auth_type"`
	Role              string               `mapstructure:"role"`
	Service           string               `mapstructure:"service"`
	Method            string               `mapstructure:"method"`
	Params            []ParamConfig        `mapstructure:"params"`
	RateLimit         RouteRateLimitConfig `mapstructure:"rate_limit"`
	Description       string               `mapstructure:"description"`
	ResponseStructure map[string]string    `mapstructure:"response_structure"`
}

// ParamConfig defines the structure for route parameters
type ParamConfig struct {
	Name        string `mapstructure:"name"`
	Type        string `mapstructure:"type"`
	Required    bool   `mapstructure:"required"` // Defaults to false
	Description string `mapstructure:"description"`
	Pattern     string `mapstructure:"pattern"`
}

// RouteRateLimitConfig holds per-route rate limit settings.
type RouteRateLimitConfig struct {
	Limit int `mapstructure:"limit"`
	Burst int `mapstructure:"burst"`
}

// LoadRouteConfigs loads and returns the route configurations from the YAML file.
func LoadRouteConfigs(cfg *config.Config) ([]RouteConfig, error) {
	viper.SetConfigName("routes")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read routes config: %w", err)
	}

	var routes []RouteConfig
	if err := viper.UnmarshalKey("routes", &routes); err != nil {
		return nil, fmt.Errorf("failed to parse routes config: %w", err)
	}

	// Set default rate limits if not defined
	for i := range routes {
		if routes[i].RateLimit.Limit == 0 {
			routes[i].RateLimit.Limit = cfg.RateLimit.DefaultLimit
		}
		if routes[i].RateLimit.Burst == 0 {
			routes[i].RateLimit.Burst = cfg.RateLimit.DefaultBurst
		}
	}

	return routes, nil
}

// SetupRoutes loads the routes from the configuration and sets them up in Gin
func SetupRoutes(r *gin.Engine, rpcClientPool *rpc.RPCClientPool, cfg *config.Config, routeConfigs []RouteConfig, logger *logging.Logger) error {

	// Set trusted proxies based on the configuration
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Fatalf("Failed to set trusted proxies: %v", err)
	}

	// Conditionally add health route
	if cfg.HealthRouteEnabled {
		r.GET("/health", func(c *gin.Context) {
			handlers.HealthHandler(c, rpcClientPool)
		})
	}

	// Conditionally add status route
	if cfg.StatusRouteEnabled {
		r.GET("/status", func(c *gin.Context) {
			handlers.StatusHandler(c, rpcClientPool)
		})
	}

	// Conditionally add JWT routes if SelfJWTEnabled
	if cfg.SelfJWTEnabled {
		r.POST("/jwt/login", handlers.JWTLoginHandler(cfg))
		r.POST("/jwt/renew", handlers.JWTRenewalHandler(cfg))
	}

	// Apply global middlewares to the router
	addMiddlewareStack(r, cfg, logger)

	// Register the routes with middlewares
	for _, routeConfig := range routeConfigs {
		// Build the middleware stack
		mws := buildMiddlewareStack(r, routeConfig, cfg)

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

// addMiddlewareStack creates and adds a global middleware stack based on the configuration
func addMiddlewareStack(r *gin.Engine, cfg *config.Config, logger *logging.Logger) {
	// Apply security headers if enabled
	if cfg.EnableSecurityHeaders {
		r.Use(middleware.SecurityHeadersMiddleware(cfg.TrustedOrigins))
	}

	// Apply CORS middleware if enabled
	if cfg.EnableCORS {
		r.Use(middleware.CORSMiddleware(cfg.TrustedOrigins))
	}

	// Apply Cloudflare headers middleware if enabled
	if cfg.EnableCloudflare {
		r.Use(middleware.CloudflareMiddleware(logger))
	}

	// Apply RBAC middleware if enabled
	//if cfg.EnableRBAC {
	//    r.Use(middleware.RBACMiddleware())
	//}
}

// buildMiddlewareStack creates the middleware stack for a given route
func buildMiddlewareStack(r *gin.Engine, route RouteConfig, cfg *config.Config) []gin.HandlerFunc {
	mws := []gin.HandlerFunc{} // Middleware stack

	if route.RateLimit.Limit == 0 {
		route.RateLimit.Limit = cfg.RateLimit.DefaultLimit
	}
	if route.RateLimit.Burst == 0 {
		route.RateLimit.Burst = cfg.RateLimit.DefaultBurst
	}

	if cfg.RateLimit.Enabled {
		mws = append(mws, middleware.RateLimitMiddleware(route.Path, route.RateLimit.Limit, route.RateLimit.Burst))
	}
	// Add authentication middleware based on auth_type
	if route.Authorization {
		switch route.AuthType {
		case "jwt":
			mws = append(mws, middleware.JWTAuthMiddleware(cfg))
		case "oauth":
			mws = append(mws, middleware.OAuthMiddleware(cfg))
		case "cloudflare_jwt":
			mws = append(mws, middleware.CloudflareJWTMiddleware(cfg))
		}
	}

	// Add RBAC middleware if a role is specified
	if route.Role != "" && cfg.EnableRBAC {
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
		//rpcClient := rpcClientPool.GetClient()
		rpcClient, err := rpcClientPool.GetClient(5 * time.Second)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "all clients are busy"})
			return
		}
		defer rpcClientPool.ReturnClient(rpcClient) // Ensure client is returned to the pool

		// Send the RPC request and get the response
		log.Printf("To call RPC: s:%v m:%v a:%v", service, method, args)
		response, err := rpcClient.CallRPC(service, method, args)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Assuming `response` is of type map[string]interface{}
		innerResponse, ok := response["response"].(map[string]interface{})
		if !ok {
			// Handle the case where "response" field is missing or not of expected type
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected response structure"})
			return
		}

		c.JSON(http.StatusOK, innerResponse)

		// Return the response to the client
		//c.JSON(200, response.Response)
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
			switch param.Type {
			case "string":
				// Validate against the pattern if one is provided
				value, ok := paramValue.(string)
				if !ok {
					return nil, fmt.Errorf("unable to parse parameter %s: - %s", param.Name, generateDescription(param))
				}
				if param.Pattern != "" {
					matched, err := regexp.MatchString(param.Pattern, value)
					if err != nil || !matched {
						return nil, fmt.Errorf("invalid parameter value for %s: does not match pattern %s - %s", param.Name, param.Pattern, generateDescription(param))
					}
				}

			case "integer":
				// Handle int conversion for both string and numeric input
				intValue, err := convertToInt(paramValue)
				if err != nil {
					return nil, fmt.Errorf("invalid parameter type for %s: expected int - %s", param.Name, generateDescription(param))
				}
				args[param.Name] = intValue

			case "number":
				// Handle float conversion for both string and numeric input
				floatValue, err := convertToFloat(paramValue)
				if err != nil {
					return nil, fmt.Errorf("invalid parameter type for %s: expected float - %s", param.Name, generateDescription(param))
				}
				args[param.Name] = floatValue

			case "boolean":
				// Handle boolean conversion for both string and native bool
				boolValue, err := convertToBool(paramValue)
				if err != nil {
					return nil, fmt.Errorf("invalid parameter type for %s: expected bool - %s", param.Name, generateDescription(param))
				}
				args[param.Name] = boolValue
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

// Helper function to convert to int
func convertToInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("unsupported type for int conversion: %v", reflect.TypeOf(value))
	}
}

// Helper function to convert to float
func convertToFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("unsupported type for float conversion: %v", reflect.TypeOf(value))
	}
}

// Helper function to convert to bool
func convertToBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("unsupported type for bool conversion: %v", reflect.TypeOf(value))
	}
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
