package cache

import (
	"context"
	"errors"
	"time"

	"golang.org/x/sync/singleflight"
)

// ComputeFunc is a function that computes the value when cache misses occur
type ComputeFunc[V any] func(ctx context.Context, key string) (V, error)

// TieredCache implements a multi-tier caching strategy
// Strategy: Local Cache → Remote Cache
// Uses singleflight to prevent cache stampede on compute function execution
type TieredCache[V any] struct {
	localCache  LocalCacher[V]
	remoteCache RemoteCacher[V]
	sfGroup     singleflight.Group
}

// NewTieredCache creates a new multi-tier cache with dependency injection
// Both localCache and remoteCache are optional (can be nil)
func NewTieredCache[V any](localCache LocalCacher[V], remoteCache RemoteCacher[V]) *TieredCache[V] {
	return &TieredCache[V]{
		localCache:  localCache,
		remoteCache: remoteCache,
	}
}

// Get retrieves a value using the tiered caching strategy with compute function:
// 1. Check local cache (L1)
// 2. Check remote cache (L2) - populate L1 on hit
// 3. Execute computeFn - populate L1 and L2 on compute
// Uses singleflight to ensure only one compute function executes per key concurrently
func (tc *TieredCache[V]) Get(ctx context.Context, key string, ttl time.Duration, computeFn ComputeFunc[V]) (V, error) {
	var zero V

	// Try to get from cache tiers
	val, found, err := tc.getCache(ctx, key)
	if err != nil {
		return zero, err
	}
	if found {
		return val, nil
	}

	// Both caches missed, execute compute function with singleflight
	result, err, _ := tc.sfGroup.Do(key, func() (interface{}, error) {
		// Double-check cache after acquiring singleflight lock
		val, found, err := tc.getCache(ctx, key)
		if err != nil {
			return zero, err
		}
		if found {
			return val, nil
		}

		// Execute compute function
		val, err = computeFn(ctx, key)
		if err != nil {
			return zero, err
		}

		// Set in caches
		if err := tc.setCache(ctx, key, val, ttl); err != nil {
			return zero, err
		}

		return val, nil
	})

	if err != nil {
		return zero, err
	}

	return result.(V), nil
}

// getCache attempts to retrieve a value from cache tiers
// Returns (value, found, error)
func (tc *TieredCache[V]) getCache(ctx context.Context, key string) (V, bool, error) {
	var zero V

	// Try local cache first (L1)
	if tc.localCache != nil {
		val, err := tc.localCache.Get(ctx, key)
		if err == nil {
			return val, true, nil
		}
		if !errors.Is(err, ErrCacheMiss) {
			return zero, false, err
		}
	}

	// Try remote cache (L2)
	if tc.remoteCache != nil {
		val, err := tc.remoteCache.Get(ctx, key)
		if err == nil {
			// TODO: Populate L1 on L2 hit
			return val, true, nil
		}
		if !errors.Is(err, ErrCacheMiss) {
			return zero, false, err
		}
	}

	// Not found in any cache
	return zero, false, nil
}

// setCache writes a value to all cache tiers
func (tc *TieredCache[V]) setCache(ctx context.Context, key string, value V, ttl time.Duration) error {
	// Set in local cache (L1)
	if tc.localCache != nil {
		if err := tc.localCache.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}
	// Set in remote cache (L2)
	if tc.remoteCache != nil {
		if err := tc.remoteCache.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

// Set stores a value in all cache tiers
func (tc *TieredCache[V]) Set(ctx context.Context, key string, value V, ttl time.Duration) error {
	return tc.setCache(ctx, key, value, ttl)
}

// Delete removes a key from all cache tiers
func (tc *TieredCache[V]) Delete(ctx context.Context, key string) error {
	if tc.localCache != nil {
		if err := tc.localCache.Delete(ctx, key); err != nil && !errors.Is(err, ErrCacheMiss) {
			return err
		}
	}
	if tc.remoteCache != nil {
		if err := tc.remoteCache.Delete(ctx, key); err != nil && !errors.Is(err, ErrCacheMiss) {
			return err
		}
	}
	return nil
}
