package rpc

import (
	"caaspay-api-go/internal/broker"
	"caaspay-api-go/internal/logging"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

// RPCClientPool manages a pool of RPC clients with active request limits and auto-scaling capabilities
type RPCClientPool struct {
	clients              []*RPCClient
	activeRequests       map[*RPCClient]int
	maxRequestsPerClient int
	initialClients       int
	maxClients           int
	broker               broker.Broker
	mutex                sync.Mutex
	scalingDown          bool
	monitorInterval      time.Duration
	logger               *logging.Logger
	ctx                  context.Context
}

// NewRPCClientPool initializes a new pool of RPC clients with configurable limits and monitoring interval
func NewRPCClientPool(ctx context.Context, initialClients, maxClients, maxRequestsPerClient int, broker broker.Broker, monitorInterval time.Duration, logger *logging.Logger) *RPCClientPool {
	pool := &RPCClientPool{
		clients:              make([]*RPCClient, 0, initialClients),
		activeRequests:       make(map[*RPCClient]int),
		maxRequestsPerClient: maxRequestsPerClient,
		initialClients:       initialClients,
		maxClients:           maxClients,
		broker:               broker,
		monitorInterval:      monitorInterval,
		logger:               logger,
		ctx:                  ctx,
	}

	// Initialize the initial pool of clients
	for i := 0; i < initialClients; i++ {
		client := NewRPCClient(broker, ctx)
		if err := client.Start(); err == nil {
			pool.clients = append(pool.clients, client)
			pool.activeRequests[client] = 0
			logger.LogAndRecord(logrus.InfoLevel, "Added Client to pool", "client_pool_scale_up", map[string]string{"client_count": fmt.Sprintf("%d", 1)})
		}
	}

	// Start monitoring and scaling down if needed
	go pool.monitorAndScaleDown()
	return pool
}

// GetClient retrieves a client from the pool, creating a new one if all are busy
func (p *RPCClientPool) GetClient(timeout time.Duration) (*RPCClient, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Find an available client with active requests below the limit
	for _, client := range p.clients {
		if p.activeRequests[client] < p.maxRequestsPerClient {
			p.activeRequests[client]++
			return client, nil
		}
	}

	// Create a new client if all are busy and maxClients limit is not reached
	if len(p.clients) < p.maxClients {
		newClient := NewRPCClient(p.broker, p.ctx)
		if err := newClient.Start(); err == nil {
			p.clients = append(p.clients, newClient)
			p.activeRequests[newClient] = 1
			p.logger.LogAndRecord(logrus.InfoLevel, "Added Client to pool", "client_pool_scale_up", map[string]string{"client_count": fmt.Sprintf("%d", 1)})
			return newClient, nil
		}
	}

	// Wait until a client is available or timeout
	waitChan := make(chan *RPCClient)
	go func() {
		for {
			time.Sleep(10 * time.Millisecond)
			p.mutex.Lock()
			for _, client := range p.clients {
				if p.activeRequests[client] < p.maxRequestsPerClient {
					p.activeRequests[client]++
					p.mutex.Unlock()
					waitChan <- client
					return
				}
			}
			p.mutex.Unlock()
		}
	}()

	select {
	case client := <-waitChan:
		return client, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout: no available clients")
	}
}

// ReturnClient releases a client back to the pool after a request completes
func (p *RPCClientPool) ReturnClient(client *RPCClient) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.activeRequests[client] > 0 {
		p.activeRequests[client]--
	}
}

func (p *RPCClientPool) monitorAndScaleDown() {
	ticker := time.NewTicker(p.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mutex.Lock()
			if len(p.clients) > p.initialClients {
				idleCount := 0
				for i := len(p.clients) - 1; i >= p.initialClients; i-- {
					client := p.clients[i]
					p.logger.LogAndRecord(logrus.InfoLevel, "Broker clients Load", "client_pool_count", map[string]string{"count": fmt.Sprintf("%d", p.activeRequests[client]), "client": fmt.Sprintf("%v", client.Whoami)})
					if p.activeRequests[client] == 0 {
						//client.Close()
						// Call the unified Close method
						if err := client.Close(); err != nil {
							p.logger.LogAndRecord(logrus.WarnLevel, "Failed to close client", "client_pool_stop_fail", map[string]string{"client": fmt.Sprintf("%v", client), "err": fmt.Sprintf("%v", err)})
						}
						p.clients = p.clients[:i]
						delete(p.activeRequests, client)
						idleCount++
					}
				}
				if idleCount > 0 {
					p.logger.LogAndRecord(logrus.InfoLevel, "Scaled down idle clients", "client_pool_scale_down", map[string]string{"idle_count": fmt.Sprintf("%d", idleCount)})
				}
			}
			p.mutex.Unlock()
		case <-p.ctx.Done():
			return
		}
	}
}

// ActiveClientCount returns the current number of active clients in the pool
func (p *RPCClientPool) ActiveClientCount() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return len(p.clients)
}

// Close closes all clients in the pool
func (p *RPCClientPool) Close() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for _, client := range p.clients {
		client.Close()
	}
}
