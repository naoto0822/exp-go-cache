package memoizer

import (
	"context"
	"errors"
	"time"
)

// ComputeFunc is a function that computes the value when cache misses occur
type ComputeFunc[V any] func(ctx context.Context, key string) (V, error)

// TieredMemoizer implements a multi-tier caching strategy
// Strategy: Local Cache → Remote Cache
type TieredMemoizer[V any] struct {
	localCache  LocalCacher[V]
	remoteCache RemoteCacher[V]
}

// NewTieredMemoizer creates a new multi-tier memoizer with dependency injection
// Both localCache and remoteCache are optional (can be nil)
func NewTieredMemoizer[V any](localCache LocalCacher[V], remoteCache RemoteCacher[V]) *TieredMemoizer[V] {
	return &TieredMemoizer[V]{
		localCache:  localCache,
		remoteCache: remoteCache,
	}
}

// Get retrieves a value using the tiered caching strategy with compute function:
// 1. Check local cache (L1)
// 2. Check remote cache (L2) - populate L1 on hit
// 3. Execute computeFn - populate L1 and L2 on compute
func (tm *TieredMemoizer[V]) Get(ctx context.Context, key string, ttl time.Duration, computeFn ComputeFunc[V]) (V, error) {
	var zero V

	// Try to get from cache tiers
	val, found, err := tm.getCache(ctx, key)
	if err != nil {
		return zero, err
	}
	if found {
		return val, nil
	}

	// Both caches missed, execute compute function
	val, err = computeFn(ctx, key)
	if err != nil {
		return zero, err
	}

	// Set in caches
	if err := tm.setCache(ctx, key, val, ttl); err != nil {
		return zero, err
	}

	return val, nil
}

// getCache attempts to retrieve a value from cache tiers
// Returns (value, found, error)
func (tm *TieredMemoizer[V]) getCache(ctx context.Context, key string) (V, bool, error) {
	var zero V

	// Try local cache first (L1)
	if tm.localCache != nil {
		val, err := tm.localCache.Get(ctx, key)
		if err == nil {
			return val, true, nil
		}
		if !errors.Is(err, ErrCacheMiss) {
			return zero, false, err
		}
	}

	// Try remote cache (L2)
	if tm.remoteCache != nil {
		val, err := tm.remoteCache.Get(ctx, key)
		if err == nil {
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
func (tm *TieredMemoizer[V]) setCache(ctx context.Context, key string, value V, ttl time.Duration) error {
	// Set in local cache (L1)
	if tm.localCache != nil {
		if err := tm.localCache.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}
	// Set in remote cache (L2)
	if tm.remoteCache != nil {
		if err := tm.remoteCache.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

// Set stores a value in all cache tiers
func (tm *TieredMemoizer[V]) Set(ctx context.Context, key string, value V, ttl time.Duration) error {
	return tm.setCache(ctx, key, value, ttl)
}

// Delete removes a key from all cache tiers
func (tm *TieredMemoizer[V]) Delete(ctx context.Context, key string) error {
	if tm.localCache != nil {
		if err := tm.localCache.Delete(ctx, key); err != nil && !errors.Is(err, ErrCacheMiss) {
			return err
		}
	}
	if tm.remoteCache != nil {
		if err := tm.remoteCache.Delete(ctx, key); err != nil && !errors.Is(err, ErrCacheMiss) {
			return err
		}
	}
	return nil
}
