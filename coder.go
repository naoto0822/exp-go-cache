package cacher

// Coder defines the interface for encoding and decoding values
type Coder[V any] interface {
	// Encode serializes a value to bytes
	Encode(value V) ([]byte, error)

	// Decode deserializes bytes to a value
	Decode(data []byte) (V, error)
}
