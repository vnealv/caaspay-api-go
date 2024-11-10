package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

// RedisBroker handles Redis operations
type RedisBroker struct {
	client    redis.UniversalClient // UniversalClient can support both Redis and Redis Cluster
	prefix    string
	isCluster bool
	pubsubs   map[string]*redis.PubSub
}

// RedisOptions encapsulates options for both standalone and cluster modes.
type RedisOptions struct {
	Addrs     []string // Addresses for cluster or single node
	Password  string
	DB        int
	Prefix    string
	IsCluster bool // Whether to use cluster mode
}

// NewRedisBroker creates a new RedisBroker instance that supports both Redis instance and Redis cluster.
func NewRedisBroker(opts RedisOptions) *RedisBroker {
	var client redis.UniversalClient
	if opts.IsCluster {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    opts.Addrs,
			Password: opts.Password,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     opts.Addrs[0], // Single node uses just one address
			Password: opts.Password,
			DB:       opts.DB,
		})
	}

	return &RedisBroker{
		client:    client,
		prefix:    opts.Prefix,
		isCluster: opts.IsCluster,
		pubsubs:   make(map[string]*redis.PubSub),
	}
}

// --------- Pub/Sub Operations ---------

// Publish sends a message to a Redis channel
func (r *RedisBroker) Publish(channel, message string) error {
	return r.client.Publish(ctx, r.applyPrefix(channel), message).Err()
}

// Subscribe subscribes to a Redis channel
//func (r *RedisBroker) Subscribe(channel string) *redis.PubSub {
//    return r.client.Subscribe(ctx, r.applyPrefix(channel))
//}

func (r *RedisBroker) Subscribe(ctx context.Context, channel string, onMessage func(map[string]interface{})) error {
	pubsub := r.client.Subscribe(ctx, r.applyPrefix(channel))

	// Wait for the subscription to be established
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to channel %s: %v", r.applyPrefix(channel), err)
	}

	// Store the PubSub instance for unsubscription
	r.pubsubs[channel] = pubsub

	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			var response map[string]interface{}

			// Parse the message payload as JSON
			if err := json.Unmarshal([]byte(msg.Payload), &response); err == nil {
				onMessage(response)
			}
		}
	}()

	return nil
}

func (r *RedisBroker) Unsubscribe(ctx context.Context, channel string) error {
	pubsub, exists := r.pubsubs[channel]
	if !exists {
		return fmt.Errorf("no subscription found for channel %s", channel)
	}

	// Unsubscribe and remove the entry from the map
	if err := pubsub.Unsubscribe(ctx); err != nil {
		return fmt.Errorf("failed to unsubscribe from channel %s: %v", channel, err)
	}
	delete(r.pubsubs, channel)
	return nil
}

// --------- Stream Operations ---------

// XAdd adds a message to a Redis stream
//func (r *RedisBroker) XAdd(stream string, values map[string]interface{}) (string, error) {
//    result, err := r.client.XAdd(ctx, &redis.XAddArgs{
//        Stream: r.applyPrefix(stream),
//        Values: values,
//    }).Result()
//    if err != nil {
//        return "", err
//    }
//    return result, nil
//}

// XAddRPC sends an RPC request to a Redis stream
//func (r *RedisBroker) XAddRPC(service, request *RPCMessage) (string, error) {
//    streamName := fmt.Sprintf("service.%s.rpc/%s", service, request.RPC)
//    messageID, err := r.client.XAdd(ctx, &redis.XAddArgs{
//        Stream: r.applyPrefix(streamName),
//        Values: map[string]interface{}{
//            "rpc":        request.RPC,
//            "who":        request.Who,
//            "message_id": request.MessageID,
//            "deadline":   request.Deadline,
//            "args":       request.Args,
//        },
//    }).Result()
//    return messageID, err
//}

// XAdd publishes a message to a Redis stream
//
//	func (r *RedisBroker) XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
//	   // Add the message to the stream
//	   messageID, err := r.client.XAdd(ctx, &redis.XAddArgs{
//	       Stream: r.applyPrefix(stream),
//	       Values: values,
//	   }).Result()
//	   if err != nil {
//	       return "", fmt.Errorf("failed to add message to stream %s: %v", stream, err)
//	   }
//	   return messageID, nil
//	}
//
// XAdd publishes a message to a Redis stream, converting complex values to JSON strings
func (r *RedisBroker) XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
	// Convert values to a format Redis accepts
	formattedValues := make(map[string]interface{})
	for k, v := range values {
		switch v := v.(type) {
		case string:
			formattedValues[k] = v
		default:
			// Marshal complex types to JSON
			jsonValue, err := json.Marshal(v)
			if err != nil {
				return "", fmt.Errorf("failed to marshal value for key %s: %v", k, err)
			}
			formattedValues[k] = string(jsonValue)
		}
	}

	// Add the formatted message to the stream
	messageID, err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: r.applyPrefix(stream),
		Values: formattedValues,
	}).Result()
	if err != nil {
		return "", fmt.Errorf("failed to add message to stream %s: %v", stream, err)
	}
	return messageID, nil
}

// XReadGroup reads from a Redis stream as a consumer group
func (r *RedisBroker) XReadGroup(stream, group, consumer string, count int64, block time.Duration, startID string) ([]redis.XStream, error) {
	streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{r.applyPrefix(stream), startID},
		Count:    count,
		Block:    block,
	}).Result()
	if err != nil {
		return nil, err
	}
	return streams, nil
}

// XAck acknowledges a message in a Redis stream
func (r *RedisBroker) XAck(stream, group string, messageIDs ...string) (int64, error) {
	count, err := r.client.XAck(ctx, r.applyPrefix(stream), group, messageIDs...).Result()
	if err != nil {
		return 0, err
	}
	return count, nil
}

// XTrim trims the stream to a specified length
func (r *RedisBroker) XTrim(stream string, maxLen int64) error {
	return r.client.XTrimMaxLen(ctx, r.applyPrefix(stream), maxLen).Err()
}

// --------- Other Redis Data Structures ---------

// Set adds a key-value pair to Redis with an optional expiration time
func (r *RedisBroker) Set(key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, r.applyPrefix(key), value, expiration).Err()
}

// Get retrieves a value from Redis by key
func (r *RedisBroker) Get(key string) (string, error) {
	result, err := r.client.Get(ctx, r.applyPrefix(key)).Result()
	if err != nil {
		return "", err
	}
	return result, nil
}

// HSet sets a field in a Redis hash
func (r *RedisBroker) HSet(key, field string, value interface{}) error {
	return r.client.HSet(ctx, r.applyPrefix(key), field, value).Err()
}

// HGet gets a field value from a Redis hash
func (r *RedisBroker) HGet(key, field string) (string, error) {
	result, err := r.client.HGet(ctx, r.applyPrefix(key), field).Result()
	if err != nil {
		return "", err
	}
	return result, nil
}

// SAdd adds a value to a Redis set
func (r *RedisBroker) SAdd(key string, values ...interface{}) error {
	return r.client.SAdd(ctx, r.applyPrefix(key), values...).Err()
}

// SMembers retrieves all members of a Redis set
func (r *RedisBroker) SMembers(key string) ([]string, error) {
	result, err := r.client.SMembers(ctx, r.applyPrefix(key)).Result()
	if err != nil {
		return nil, err
	}
	return result, nil
}

// LPush adds a value to a Redis list (left push)
func (r *RedisBroker) LPush(key string, values ...interface{}) error {
	return r.client.LPush(ctx, r.applyPrefix(key), values...).Err()
}

// RPop pops a value from a Redis list (right pop)
func (r *RedisBroker) RPop(key string) (string, error) {
	result, err := r.client.RPop(ctx, r.applyPrefix(key)).Result()
	if err != nil {
		return "", err
	}
	return result, nil
}

// --------- Utility Functions ---------

// NewMessage creates a new Redis-specific RPC message
//func (r *RedisBroker) NewMessage(rpc, who string, args map[string]interface{}, timeout time.Duration) *RPCMessage {
//    return &RPCMessage{
//        RPC:       rpc,
//        MessageID: uuid.New().String(),
//        Who:       who,
//        Deadline:  time.Now().Add(timeout).Unix(),
//        Args:      args,
//    }
//}

// GenerateUUID generates a UUID
func (r *RedisBroker) GenerateUUID() string {
	return uuid.New().String()
}

// Apply prefix to a Redis key
func (r *RedisBroker) applyPrefix(key string) string {
	return r.prefix + "." + key
	//return key
}

// Close closes the Redis connection
func (r *RedisBroker) Close() error {
	return r.client.Close()
}
