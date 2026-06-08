package stratum

import (
	"encoding/json"
)

// JSONRPC represents a standard Stratum message
type JSONRPC struct {
	ID     interface{}     `json:"id"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  interface{}     `json:"error,omitempty"`
}

// Parse parses a byte slice into a JSONRPC message
func Parse(data []byte) (*JSONRPC, error) {
	var msg JSONRPC
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// Marshal converts a JSONRPC message back to JSON
func (msg *JSONRPC) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}

// Common Stratum methods
const (
	MethodMiningSubscribe     = "mining.subscribe"
	MethodMiningAuthorize     = "mining.authorize"
	MethodMiningSubmit        = "mining.submit"
	MethodMiningSetDifficulty = "mining.set_difficulty"
	MethodMiningNotify        = "mining.notify"
)
