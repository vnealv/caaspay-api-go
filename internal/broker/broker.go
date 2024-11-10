package broker

import (
	"context"
)

// Broker defines the interface that any message broker must implement
type Broker interface {
	Subscribe(ctx context.Context, channel string, onMessage func(map[string]interface{})) error
	Unsubscribe(ctx context.Context, channel string) error
	XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error)
	GenerateUUID() string
	Close() error
}
