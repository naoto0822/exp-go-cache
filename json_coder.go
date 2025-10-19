package cache

import "encoding/json"

// JSONCoder implements Coder using JSON encoding
type JSONCoder[V any] struct{}

// NewJSONCoder creates a new JSONCoder instance
func NewJSONCoder[V any]() *JSONCoder[V] {
	return &JSONCoder[V]{}
}

// Encode serializes a value to JSON bytes
func (c *JSONCoder[V]) Encode(value V) ([]byte, error) {
	return json.Marshal(value)
}

// Decode deserializes JSON bytes to a value
func (c *JSONCoder[V]) Decode(data []byte) (V, error) {
	var value V
	err := json.Unmarshal(data, &value)
	return value, err
}
