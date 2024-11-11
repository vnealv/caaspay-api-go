package rpc

import (
	"caaspay-api-go/internal/broker"
	"context"
	"fmt"
	"time"
)

// RPCClient handles sending and receiving RPC messages using a broker
type RPCClient struct {
	broker     broker.Broker
	pending    map[string]chan *RPCMessage
	Whoami     string
	Subscribed bool
	ctx        context.Context
}

// NewRPCClient creates a new instance of RPCClient using the provided broker
func NewRPCClient(broker broker.Broker, ctx context.Context) *RPCClient {
	return &RPCClient{
		broker:  broker,
		Whoami:  broker.GenerateUUID(),
		pending: make(map[string]chan *RPCMessage),
		ctx:     ctx,
	}
}

// Start subscribes to the UUID channel for receiving responses
func (c *RPCClient) Start() error {
	err := c.broker.Subscribe(c.ctx, c.Whoami, func(msg map[string]interface{}) {
		message := MapToRPCMessage(msg)
		if pendingChan, exists := c.pending[message.MessageID]; exists {
			pendingChan <- message
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	c.Subscribed = true
	return nil
}

// CallRPC sends an RPC message and waits for the response
// func (c *RPCClient) CallRPC(service, method string, args map[string]interface{}, timeout time.Duration) (*RPCMessage, error) {
func (c *RPCClient) CallRPC(service, method string, args map[string]interface{}, timeout ...time.Duration) (map[string]interface{}, error) {
	if !c.Subscribed {
		return nil, fmt.Errorf("client is not subscribed to channel")
	}

	// Set default timeout if none is provided
	effectiveTimeout := 120 * time.Second
	if len(timeout) > 0 && timeout[0] > 0 {
		effectiveTimeout = timeout[0]
	}

	request := NewRPCMessage(method, c.Whoami, args, effectiveTimeout)
	messageID := request.MessageID
	respChan := make(chan *RPCMessage, 1)
	c.pending[messageID] = respChan

	// Send the message to Redis via XAdd
	// myriad.service.control.authentication.login.rpc/login
	streamName := fmt.Sprintf("service.%s.rpc/%s", service, request.RPC)
	if _, err := c.broker.XAdd(c.ctx, streamName, request.ToMap()); err != nil {
		return nil, err
	}

	select {
	case resp := <-respChan:
		delete(c.pending, messageID)
		return resp.Response, nil // Return only the response field
	case <-time.After(effectiveTimeout):
		delete(c.pending, messageID)
		return nil, fmt.Errorf("rpc call timeout")
	}
}

// Stop unsubscribes from the UUID channel and marks the client as unsubscribed
//func (c *RPCClient) Stop() error {
//    if c.Subscribed {
//        err := c.broker.Unsubscribe(c.ctx, c.Whoami)
//        if err != nil {
//            return fmt.Errorf("failed to unsubscribe: %w", err)
//        }
//        c.Subscribed = false
//    }
//    return nil
//}

// Close closes the RPC client
func (c *RPCClient) Close() error {
	//    return c.broker.Close()
	// Unsubscribe if currently subscribed
	if c.Subscribed {
		if err := c.broker.Unsubscribe(c.ctx, c.Whoami); err != nil {
			return fmt.Errorf("failed to unsubscribe: %w", err)
		}
		c.Subscribed = false
	}

	// Close broker connection to clean up resources
	//if err := c.broker.Close(); err != nil {
	//	return fmt.Errorf("failed to close broker connection: %w", err)
	//}
	return nil
}

func MapToRPCMessage(data map[string]interface{}) *RPCMessage {
	message := &RPCMessage{}
	message.FromMap(data)
	return message
}
