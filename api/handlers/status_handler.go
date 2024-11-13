package handlers

import (
	"caaspay-api-go/internal/rpc"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// HealthHandler checks if the RPC client pool is available and returns API health status.
func HealthHandler(c *gin.Context, rpcClientPool *rpc.RPCClientPool) {
	client, err := rpcClientPool.GetClient(2 * time.Second) // Check client availability with a timeout
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "reason": "RPC client unavailable"})
		return
	}
	rpcClientPool.ReturnClient(client) // Return the client to the pool if acquired
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// StatusHandler returns the status of the application based on internal checks.
func StatusHandler(c *gin.Context, rpcClientPool *rpc.RPCClientPool) {
	client, err := rpcClientPool.GetClient(2 * time.Second)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "reason": "RPC client issue"})
		return
	}
	rpcClientPool.ReturnClient(client)
	// Add more internal checks here if necessary
	c.JSON(http.StatusOK, gin.H{"status": "operational"})
}
