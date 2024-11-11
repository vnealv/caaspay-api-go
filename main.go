package main

import (
	"caaspay-api-go/api/config"
	"caaspay-api-go/api/routes"
	"caaspay-api-go/internal/broker"
	"caaspay-api-go/internal/logging"
	"caaspay-api-go/internal/metrics"
	"caaspay-api-go/internal/rpc"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	//    r := gin.Default()

	cfg, err := config.LoadAPIConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	// Initialize metrics based on configuration
	var metricsClient logging.Metrics
	//    if appConfig.MetricsEnabled {
	if 1 < 0 {
		//metricsClient, err = metrics.NewDataDogMetrics(appConfig.DatadogAddr)
		var err error
		metricsClient, err = metrics.NewDataDogMetrics("127.0.0.1:8125")
		if err != nil {
			log.Fatalf("Failed to initialize Datadog metrics: %v", err)
		}
	}

	//logger := logging.NewLogger("caaspay-service", "development", appConfig.MetricsEnabled, metricsClient)
	logger := logging.NewLogger("caaspay-service", "development", false, metricsClient)

	// Set up Gin with logger middleware
	// gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		logger.Middleware()(c)
	})

	// Initialize Redis broker with options
	redisOptions := broker.RedisOptions{
		Addrs:     []string{"codensmoke-support-redis-single-1:6379"}, // Single node address
		Prefix:    cfg.Redis.Prefix,
		IsCluster: cfg.Redis.IsCluster, // Set to true if you want to use a Redis cluster
	}
	redisBroker := broker.NewRedisBroker(redisOptions)

	// Set up a context for the RPC clients
	ctx := context.Background()

	// Initialize the RPC client pool using the Redis broker
	rpcClientPool := rpc.NewRPCClientPool(ctx, 4, 10, 2, redisBroker, 5*time.Second, logger)
	fmt.Fprintln(os.Stdout, "This is written directly to stdout")

	// Initialize the routes with the route configuration
	if err := routes.SetupRoutes(r, rpcClientPool); err != nil {
		//log.Fatalf("Failed to set up routes: %v", err)
		logger.LogAndRecord(logrus.ErrorLevel, "Failed to set up routes", "setup_routes_error", map[string]string{"error": fmt.Sprintf("err %v", err)})
	}

	// Start the API server
	if err := r.Run(fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}

	// Close the broker and RPC client pool when the server shuts down
	defer redisBroker.Close()
	defer rpcClientPool.Close()
}
