package main

import (
	"caaspay-api-go/api/routes"
	"caaspay-api-go/internal/rpc"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Initialize the RPC client pool
	rpcClientPool := rpc.NewRPCClientPool("codensmoke-support-redis-single-1:6379", 30*time.Second, 5)
	fmt.Fprintln(os.Stdout, "This is written directly to stdout")

	// Initialize the routes using the route config
	err := routes.SetupRoutes(r, rpcClientPool, "config/routes.yaml")
	if err != nil {
		log.Fatalf("Failed to set up routes: %v", err)
	}

	// Start the API server
	r.Run(":8080")
}
