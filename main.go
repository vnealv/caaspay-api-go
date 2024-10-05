package main

import (
	"log"
	"github.com/gin-gonic/gin"
	"caaspay-api-go/api/middleware"
	"caaspay-api-go/api/routes"
)

func main() {
	r := gin.Default()

	// Load routes dynamically from YAML
	routeConfigs, err := routes.LoadRoutes()
	if err != nil {
		log.Fatalf("Failed to load routes: %v", err)
	}

	// Register routes dynamically
	for _, route := range routeConfigs {
		mws := []gin.HandlerFunc{} // Middleware stack for this route

		// Add authentication middleware based on auth_type
		if route.Authorization {
			switch route.AuthType {
			case "jwt":
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

		// Register route with the selected middlewares
		switch route.Type {
		case "GET":
			r.GET(route.Path, append(mws, genericHandler(route))...)
		case "POST":
			r.POST(route.Path, append(mws, genericHandler(route))...)
		default:
			log.Printf("Unsupported route type: %s for path: %s", route.Type, route.Path)
		}
	}

	r.Run(":8080") // Start the server
}

// Placeholder for your generic handler
func genericHandler(route routes.RouteConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Route " + route.Path + " handled successfully",
		})
	}
}

