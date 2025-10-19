package cache

import (
	"bytes"

	"github.com/hashicorp/go-msgpack/v2/codec"
)

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
