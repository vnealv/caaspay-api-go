package main

import (
	"caaspay-api-go/api/config"
	"caaspay-api-go/api/middleware"
	"caaspay-api-go/api/routes"
	"caaspay-api-go/internal/broker"
	"caaspay-api-go/internal/logging"
	"caaspay-api-go/internal/metrics"
	"caaspay-api-go/internal/openapi"
	"caaspay-api-go/internal/rpc"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"net/http"
)

func main() {
	//    r := gin.Default()

	ctx := context.Background()
	cfg, err := config.LoadAPIConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	// Load route configurations once
	routeConfigs, err := routes.LoadRouteConfigs(cfg)
	if err != nil {
		log.Fatalf("Failed to load route configurations: %v", err)
	}
	// Initialize Datadog metrics and OpenTelemetry tracer
	metricsClient, err := metrics.NewDataDogMetrics(cfg.DatadogAddr, "caaspay-service", cfg.Env)
	if err != nil {
		log.Fatalf("Failed to initialize Datadog metrics and tracer: %v", err)
	}
	defer metricsClient.Close()

	//logger := logging.NewLogger("caaspay-service", "development", appConfig.MetricsEnabled, metricsClient)
	logger := logging.NewLogger("caaspay-service", cfg.Env, "debug", false, metricsClient, ctx)

	// Set up Gin with logger middleware
	// gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		logger.Middleware()(c)
	})
	r.Use(otelgin.Middleware("caaspay-api-go"))
	// Secure headers and CORS: specify trusted origins for CORS
	trustedOrigins := []string{"https://yourdomain.com", "https://another-trusted-domain.com", "https://api.caaspay.com", "http://api.caaspay.com"}
	r.Use(middleware.SecurityHeadersMiddleware(trustedOrigins)) // Add security headers
	r.Use(func(c *gin.Context) {
		logger.Middleware()(c)
	})
	// Set trusted proxies based on the configuration
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Fatalf("Failed to set trusted proxies: %v", err)
	}
	r.Use(middleware.CloudflareMiddleware(logger))

	// Initialize Redis broker with options
	redisOptions := broker.RedisOptions{
		Addrs:     []string{"codensmoke-support-redis-single-1:6379"}, // Single node address
		Prefix:    cfg.Redis.Prefix,
		IsCluster: cfg.Redis.IsCluster, // Set to true if you want to use a Redis cluster
	}
	redisBroker := broker.NewRedisBroker(redisOptions)

	// Initialize the RPC client pool using the Redis broker
	rpcClientPool := rpc.NewRPCClientPool(ctx, 4, 10, 2, redisBroker, 5*time.Second, logger)
	fmt.Fprintln(os.Stdout, "This is written directly to stdout")

	// Initialize the routes with the route configuration
	if err := routes.SetupRoutes(r, rpcClientPool, cfg, routeConfigs); err != nil {
		//log.Fatalf("Failed to set up routes: %v", err)
		logger.LogWithStats("error", "Failed to set up routes", map[string]string{"metric_name": "setup_routes_error", "error": fmt.Sprintf("err %v", err)}, nil)
	}

	// Generate OpenAPI spec from routeConfigs and additional static routes
	openAPISpec, err := openapi.GenerateOpenAPISpec(routeConfigs)
	if err != nil {
		logger.LogWithStats("error", "Failed to generate OpenAPI spec", map[string]string{"error": err.Error()}, nil)
	} else {
		r.GET("/openapi.json", func(c *gin.Context) {
			c.JSON(http.StatusOK, openAPISpec)
		})
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/openapi.json")))

	// Start the API server
	if err := r.Run(fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}

	// Close the broker and RPC client pool when the server shuts down
	defer redisBroker.Close()
	defer rpcClientPool.Close()
}
