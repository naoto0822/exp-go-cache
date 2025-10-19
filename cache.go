package cacher

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCacheMiss indicates the key was not found in cache
	ErrCacheMiss = errors.New("cache miss")
)

// LocalCacher defines the interface for local cache implementations with generic type support
type LocalCacher[V any] interface {
	// Get retrieves a value from cache
	// Returns ErrCacheMiss if the key is not found
	Get(ctx context.Context, key string) (V, error)

	// Set stores a value in cache with a TTL
	Set(ctx context.Context, key string, value V, ttl time.Duration) error

	// Delete removes a value from cache
	// Returns ErrCacheMiss if the key is not found
	Delete(ctx context.Context, key string) error
}

// RemoteCacher defines the interface for remote cache implementations with generic type support
type RemoteCacher[V any] interface {
	// Get retrieves a value from cache
	// Returns ErrCacheMiss if the key is not found
	Get(ctx context.Context, key string) (V, error)

	// Set stores a value in cache with a TTL
	Set(ctx context.Context, key string, value V, ttl time.Duration) error

	// Delete removes a value from cache
	// Returns ErrCacheMiss if the key is not found
	Delete(ctx context.Context, key string) error
}

// BatchLocalCacher defines the interface for local cache implementations that support batch operations
type BatchLocalCacher[V any] interface {
	LocalCacher[V]

	// BatchGet retrieves multiple values from cache
	// Returns a map of key-value pairs for found keys
	// Missing keys are simply not included in the returned map
	BatchGet(ctx context.Context, keys []string) (map[string]V, error)

	// BatchSet stores multiple values in cache with a TTL
	// All items share the same TTL
	BatchSet(ctx context.Context, items map[string]V, ttl time.Duration) error
}

// BatchRemoteCacher defines the interface for remote cache implementations that support batch operations
type BatchRemoteCacher[V any] interface {
	RemoteCacher[V]

	// BatchGet retrieves multiple values from cache
	// Returns a map of key-value pairs for found keys
	// Missing keys are simply not included in the returned map
	BatchGet(ctx context.Context, keys []string) (map[string]V, error)

	// BatchSet stores multiple values in cache with a TTL
	// All items share the same TTL
	BatchSet(ctx context.Context, items map[string]V, ttl time.Duration) error
}
