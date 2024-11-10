package broker

import (
//	"time"
	"context"
//	"encoding/json"
//	"fmt"
)

// Broker defines the interface that any message broker must implement
//type Broker interface {
//    SubscribeRPC(channel string, onMessage func(*RPCMessage)) error
//    XAddRPC(service, request *RPCMessage) error
//    GenerateUUID() string
//    Close() error
//    NewMessage(rpc, who string, args map[string]interface{}, timeout time.Duration) *RPCMessage
//}

// Broker defines the interface that any message broker must implement
type Broker interface {
//    Subscribe(channel string, onMessage func(map[string]interface{})) error
    Subscribe(ctx context.Context, channel string, onMessage func(map[string]interface{})) error
    Unsubscribe(ctx context.Context, channel string) error
//    XAdd(channel string, message map[string]interface{}) error
	XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error)
    GenerateUUID() string
    Close() error
//    NewMessage(rpc, who string, args map[string]interface{}, timeout time.Duration) map[string]interface{}
}


