package memoizer

import (
	"bytes"
	"encoding/json"

	"github.com/hashicorp/go-msgpack/v2/codec"
)

// Coder defines the interface for encoding and decoding values
type Coder[V any] interface {
	// Encode serializes a value to bytes
	Encode(value V) ([]byte, error)

	// Decode deserializes bytes to a value
	Decode(data []byte) (V, error)
}

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

// MessagePackCoder implements Coder using MessagePack encoding
type MessagePackCoder[V any] struct {
	handle *codec.MsgpackHandle
}

// NewMessagePackCoder creates a new MessagePackCoder instance
func NewMessagePackCoder[V any]() *MessagePackCoder[V] {
	return &MessagePackCoder[V]{
		handle: &codec.MsgpackHandle{},
	}
}

// Encode serializes a value to MessagePack bytes
func (c *MessagePackCoder[V]) Encode(value V) ([]byte, error) {
	var buf bytes.Buffer
	enc := codec.NewEncoder(&buf, c.handle)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode deserializes MessagePack bytes to a value
func (c *MessagePackCoder[V]) Decode(data []byte) (V, error) {
	var value V
	dec := codec.NewDecoderBytes(data, c.handle)
	if err := dec.Decode(&value); err != nil {
		return value, err
	}
	return value, nil
}
