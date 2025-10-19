package cacher

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto"
)

// RistrettoCache wraps ristretto cache to implement the LocalCacher interface with generic type support
type RistrettoCache[V any] struct {
	cache *ristretto.Cache
}

type RistrettoCacheConfig struct {
	// NumCounters determines the number of keys tracked for admission & eviction.
	// A good starting point is 10x the number of items you expect to keep in cache.
	NumCounters int64

	// MaxCost is the maximum total cost of items in cache.
	// When cost is set to 1 per item, this effectively limits the number of items.
	MaxCost int64

	// BufferItems is the size of the Get/Set buffers.
	// A larger buffer improves throughput but uses more memory.
	BufferItems int64
}

func DefaultRistrettoCacheConfig() *RistrettoCacheConfig {
	return &RistrettoCacheConfig{
		NumCounters: 1e7,     // 10 million counters
		MaxCost:     1 << 30, // 1GB max cost
		BufferItems: 64,
	}
}

// NewRistrettoCache creates a new RistrettoCache instance
func NewRistrettoCache[V any](config *RistrettoCacheConfig) (*RistrettoCache[V], error) {
	if config == nil {
		config = DefaultRistrettoCacheConfig()
	}
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.NumCounters,
		MaxCost:     config.MaxCost,
		BufferItems: config.BufferItems,
	})
	if err != nil {
		return nil, err
	}
	return &RistrettoCache[V]{
		cache: cache,
	}, nil
}

// Get retrieves a value from the cache
func (r *RistrettoCache[V]) Get(ctx context.Context, key string) (V, error) {
	var zero V
	value, found := r.cache.Get(key)
	if !found {
		return zero, ErrCacheMiss
	}
	// Type assertion with safety check
	if v, ok := value.(V); ok {
		return v, nil
	}
	return zero, ErrCacheMiss
}

// Set stores a value in the cache with a TTL
func (r *RistrettoCache[V]) Set(ctx context.Context, key string, value V, ttl time.Duration) error {
	cost := int64(1)
	if !r.cache.SetWithTTL(key, value, cost, ttl) {
		return nil
	}
	r.cache.Wait()
	return nil
}

// Delete removes a value from the cache
func (r *RistrettoCache[V]) Delete(ctx context.Context, key string) error {
	_, found := r.cache.Get(key)
	if !found {
		return ErrCacheMiss
	}
	r.cache.Del(key)
	return nil
}

// Close closes the cache and releases resources
func (r *RistrettoCache[V]) Close() error {
	r.cache.Close()
	return nil
}

// Clear removes all items from the cache
func (r *RistrettoCache[V]) Clear() {
	r.cache.Clear()
}

// Metrics returns cache metrics from ristretto
func (r *RistrettoCache[V]) Metrics() *ristretto.Metrics {
	return r.cache.Metrics
}
