package rpc

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"
)

// RPCMessage defines the structure of an RPC message
type RPCMessage struct {
    RPC         string                 `json:"rpc"`
    MessageID   string                 `json:"message_id"`
    TransportID string                 `json:"transport_id,omitempty"`
    Who         string                 `json:"who"`
    Deadline    int64                  `json:"deadline"`
    Args        map[string]interface{} `json:"args"`
    Response    map[string]interface{} `json:"response,omitempty"`
    Stash       map[string]interface{} `json:"stash,omitempty"`
    Trace       map[string]interface{} `json:"trace,omitempty"`
}

// NewRPCMessage creates a new RPC message with a timeout deadline
func NewRPCMessage(rpc, who string, args map[string]interface{}, timeout time.Duration) *RPCMessage {
    return &RPCMessage{
        RPC:       rpc,
        MessageID: uuid.New().String(), // Generate a new UUID
        Who:       who,
        Deadline:  time.Now().Add(timeout).Unix(),
        Args:      args,
		Response:  map[string]interface{}{}, // Empty object
        Stash:     map[string]interface{}{}, // Empty object
        Trace:     map[string]interface{}{}, // Empty object
    }
}

// ToJSON serializes the RPC message to JSON
func (m *RPCMessage) ToJSON() (string, error) {
    jsonData, err := json.Marshal(m)
    if err != nil {
        return "", fmt.Errorf("failed to serialize message: %w", err)
    }
    return string(jsonData), nil
}

// FromJSON deserializes an RPC message from JSON
//func FromJSON(jsonString string) (*RPCMessage, error) {
//    var msg RPCMessage
//    err := json.Unmarshal([]byte(jsonString), &msg)
//    if err != nil {
//        return nil, fmt.Errorf("failed to deserialize message: %w", err)
//    }
//    return &msg, nil
//}

// FromJSON deserializes a JSON string into an RPCMessage
func FromJSON(jsonString string) (*RPCMessage, error) {
    var msg RPCMessage
    if err := json.Unmarshal([]byte(jsonString), &msg); err != nil {
        return nil, fmt.Errorf("failed to deserialize message: %w", err)
    }

    // Parse nested fields (Args, Response, Stash, Trace) if they are JSON strings
    msg.Args, _ = parseNestedJSON(msg.Args)
    msg.Response, _ = parseNestedJSON(msg.Response)
    msg.Stash, _ = parseNestedJSON(msg.Stash)
    msg.Trace, _ = parseNestedJSON(msg.Trace)

    return &msg, nil
}

// parseNestedJSON handles nested JSON strings within fields
func parseNestedJSON(field interface{}) (map[string]interface{}, error) {
    if str, ok := field.(string); ok {
        var nestedMap map[string]interface{}
        if err := json.Unmarshal([]byte(str), &nestedMap); err != nil {
            return nil, fmt.Errorf("failed to parse nested JSON: %w", err)
        }
        return nestedMap, nil
    }
    if fieldMap, ok := field.(map[string]interface{}); ok {
        return fieldMap, nil
    }
    return make(map[string]interface{}), nil
}


// ToMap converts the RPCMessage to a map for broker use
func (m *RPCMessage) ToMap() map[string]interface{} {
    return map[string]interface{}{
        "rpc":         m.RPC,
        "message_id":  m.MessageID,
        "transport_id": m.TransportID,
        "who":         m.Who,
        "deadline":    m.Deadline,
        "args":        m.Args,
        "response":    m.Response,
        "stash":       m.Stash,
        "trace":       m.Trace,
    }
}

func (m *RPCMessage) FromMap(data map[string]interface{}) {
    if val, ok := data["rpc"].(string); ok { m.RPC = val }
    if val, ok := data["message_id"].(string); ok { m.MessageID = val }
    if val, ok := data["transport_id"].(string); ok { m.TransportID = val }
    if val, ok := data["who"].(string); ok { m.Who = val }
    if val, ok := data["deadline"].(int64); ok { m.Deadline = val }
    if args, ok := data["args"].(map[string]interface{}); ok { m.Args = args }
    if response, ok := data["response"].(map[string]interface{}); ok { m.Response = response }
    if stash, ok := data["stash"].(map[string]interface{}); ok { m.Stash = stash }
    if trace, ok := data["trace"].(map[string]interface{}); ok { m.Trace = trace }
}


