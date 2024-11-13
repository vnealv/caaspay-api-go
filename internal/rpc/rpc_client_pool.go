package rpc

import (
	"caaspay-api-go/internal/broker"
	"caaspay-api-go/internal/logging"
	"context"
	"fmt"
	"sync"
	"time"
)

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

func NewRPCClientPool(ctx context.Context, initialClients, maxClients, maxRequestsPerClient int, broker broker.Broker, monitorInterval time.Duration, scaleDown bool, logger *logging.Logger) *RPCClientPool {
	pool := &RPCClientPool{
		clients:              make([]*RPCClient, 0, initialClients),
		activeRequests:       make(map[*RPCClient]int),
		maxRequestsPerClient: maxRequestsPerClient,
		initialClients:       initialClients,
		maxClients:           maxClients,
		broker:               broker,
		monitorInterval:      monitorInterval,
		scalingDown:          scaleDown,
		logger:               logger,
		ctx:                  ctx,
	}

	for i := 0; i < initialClients; i++ {
		client := NewRPCClient(broker, ctx)
		if err := client.Start(); err == nil {
			pool.clients = append(pool.clients, client)
			pool.activeRequests[client] = 0
			logger.LogWithStats("info", "Added Client to pool", map[string]string{
				"metric_name":  "client_pool_scale_up",
				"metric_value": fmt.Sprintf("%d", 1),
			}, nil)
		}
	}

	go pool.monitorPoolStatus()
	if scaleDown {
		go pool.scaleDownClients()
	}
	return pool
}

// Monitor the pool status: logs active client count and requests per client.
func (p *RPCClientPool) monitorPoolStatus() {
	ticker := time.NewTicker(p.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mutex.Lock()
			activeClientCount := len(p.clients)
			activeRequestsCount := 0
			for _, requests := range p.activeRequests {
				activeRequestsCount += requests
			}
			p.logger.LogWithStats("info", "Monitoring RPC Client Pool", map[string]string{
				"metric_name":         "client_pool_status",
				"active_client_count": fmt.Sprintf("%d", activeClientCount),
				"active_requests":     fmt.Sprintf("%d", activeRequestsCount),
			}, nil)
			p.mutex.Unlock()
		case <-p.ctx.Done():
			return
		}
	}
}

// Scale down the pool by removing idle clients if scaling is enabled.
func (p *RPCClientPool) scaleDownClients() {
	ticker := time.NewTicker(p.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !p.scalingDown {
				continue
			}

			p.mutex.Lock()
			if len(p.clients) > p.initialClients {
				idleCount := 0
				for i := len(p.clients) - 1; i >= p.initialClients; i-- {
					client := p.clients[i]
					if p.activeRequests[client] == 0 {
						if err := client.Close(); err == nil {
							p.clients = p.clients[:i]
							delete(p.activeRequests, client)
							idleCount++
						} else {
							p.logger.LogWithStats("warn", "Failed to close client", map[string]string{
								"metric_name": "client_pool_stop_fail",
								"client":      client.Whoami,
								"error":       fmt.Sprintf("%v", err),
							}, nil)
						}
					}
				}
				if idleCount > 0 {
					p.logger.LogWithStats("info", "Scaled down idle clients", map[string]string{
						"metric_name": "client_pool_scale_down",
						"idle_count":  fmt.Sprintf("%d", idleCount),
					}, nil)
				}
			}
			p.mutex.Unlock()
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *RPCClientPool) GetClient(timeout time.Duration) (*RPCClient, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, client := range p.clients {
		if p.activeRequests[client] < p.maxRequestsPerClient {
			p.activeRequests[client]++
			return client, nil
		}
	}

	if len(p.clients) < p.maxClients {
		newClient := NewRPCClient(p.broker, p.ctx)
		if err := newClient.Start(); err == nil {
			p.clients = append(p.clients, newClient)
			p.activeRequests[newClient] = 1
			p.logger.LogWithStats("info", "Added Client to pool", map[string]string{
				"metric_name":  "client_pool_scale_up",
				"metric_value": fmt.Sprintf("%d", 1),
			}, nil)
			return newClient, nil
		}
	}

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

func (p *RPCClientPool) ReturnClient(client *RPCClient) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.activeRequests[client] > 0 {
		p.activeRequests[client]--
	}
}

func (p *RPCClientPool) ActiveClientCount() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return len(p.clients)
}

func (p *RPCClientPool) Close() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for _, client := range p.clients {
		client.Close()
	}
}
