package cache

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCacheMiss indicates the key was not found in cache
	ErrCacheMiss = errors.New("cache miss")
)

// Cacher defines the unified interface for cache implementations (local or remote)
// This interface can be used for multi-tier caching where caches[0] is L1, caches[1] is L2, etc.
type Cacher[V any] interface {
	// Get retrieves a value from cache
	// Returns ErrCacheMiss if the key is not found
	Get(ctx context.Context, key string) (V, error)

	// Set stores a value in cache with a TTL
	Set(ctx context.Context, key string, value V, ttl time.Duration) error

	// Delete removes a value from cache
	// Returns ErrCacheMiss if the key is not found
	Delete(ctx context.Context, key string) error
}

// BatchCacher defines the interface for cache implementations that support batch operations
type BatchCacher[V any] interface {
	Cacher[V]

	// BatchGet retrieves multiple values from cache
	// Returns a map of key-value pairs for found keys
	// Missing keys are simply not included in the returned map
	BatchGet(ctx context.Context, keys []string) (map[string]V, error)

	// BatchSet stores multiple values in cache with a TTL
	// All items share the same TTL
	BatchSet(ctx context.Context, items map[string]V, ttl time.Duration) error
}

// Deprecated: Use Cacher instead
// LocalCacher defines the interface for local cache implementations with generic type support
type LocalCacher[V any] interface {
	Cacher[V]
}

// Deprecated: Use Cacher instead
// RemoteCacher defines the interface for remote cache implementations with generic type support
type RemoteCacher[V any] interface {
	Cacher[V]
}

// Deprecated: Use BatchCacher instead
// BatchLocalCacher defines the interface for local cache implementations that support batch operations
type BatchLocalCacher[V any] interface {
	BatchCacher[V]
}

// Deprecated: Use BatchCacher instead
// BatchRemoteCacher defines the interface for remote cache implementations that support batch operations
type BatchRemoteCacher[V any] interface {
	BatchCacher[V]
}
