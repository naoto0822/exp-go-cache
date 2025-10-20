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
// Strategy: caches[0] (L1) → caches[1] (L2) → ... → caches[n] (Ln)
// Uses singleflight to prevent cache stampede on compute function execution
type TieredCache[V any] struct {
	caches  []Cacher[V]
	sfGroup singleflight.Group
}

// NewTieredCache creates a new multi-tier cache with dependency injection
// caches is a slice where caches[0] is L1 (fastest), caches[1] is L2, etc.
// Empty or nil caches in the slice are skipped
func NewTieredCache[V any](caches ...Cacher[V]) *TieredCache[V] {
	// Filter out nil caches
	validCaches := make([]Cacher[V], 0, len(caches))
	for _, cache := range caches {
		if cache != nil {
			validCaches = append(validCaches, cache)
		}
	}
	return &TieredCache[V]{
		caches: validCaches,
	}
}

// Get retrieves a value using the tiered caching strategy with compute function:
// 1. Check L1, L2, ..., Ln in order
// 2. If found in Li (i > 0), populate upper tiers (L0 to Li-1)
// 3. If not found in any tier, execute computeFn and populate all tiers
// Uses singleflight to ensure only one compute function executes per key concurrently
func (tc *TieredCache[V]) Get(ctx context.Context, key string, ttl time.Duration, computeFn ComputeFunc[V]) (V, error) {
	var zero V

	// Try to get from cache tiers
	val, tierIndex, found, err := tc.getCache(ctx, key)
	if err != nil {
		return zero, err
	}
	if found {
		// Populate upper tiers if found in L2 or below
		if tierIndex > 0 {
			_ = tc.populateUpperTiers(ctx, key, val, ttl, tierIndex)
		}
		return val, nil
	}

	// All caches missed, execute compute function with singleflight
	result, err, _ := tc.sfGroup.Do(key, func() (interface{}, error) {
		// Double-check cache after acquiring singleflight lock
		val, tierIndex, found, err := tc.getCache(ctx, key)
		if err != nil {
			return zero, err
		}
		if found {
			// Populate upper tiers if found in L2 or below
			if tierIndex > 0 {
				_ = tc.populateUpperTiers(ctx, key, val, ttl, tierIndex)
			}
			return val, nil
		}

		// Execute compute function
		val, err = computeFn(ctx, key)
		if err != nil {
			return zero, err
		}

		// Set in all caches
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
// Returns (value, tierIndex, found, error)
// tierIndex indicates which tier the value was found in (0 = L1, 1 = L2, etc.)
func (tc *TieredCache[V]) getCache(ctx context.Context, key string) (V, int, bool, error) {
	var zero V

	// Try each cache tier in order
	for i, cache := range tc.caches {
		val, err := cache.Get(ctx, key)
		if err == nil {
			return val, i, true, nil
		}
		if !errors.Is(err, ErrCacheMiss) {
			return zero, -1, false, err
		}
	}

	// Not found in any cache
	return zero, -1, false, nil
}

// populateUpperTiers writes a value to all cache tiers above the specified tier
// Used when a value is found in L2+ to populate L1
func (tc *TieredCache[V]) populateUpperTiers(ctx context.Context, key string, value V, ttl time.Duration, foundTierIndex int) error {
	for i := 0; i < foundTierIndex && i < len(tc.caches); i++ {
		if err := tc.caches[i].Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

// setCache writes a value to all cache tiers
func (tc *TieredCache[V]) setCache(ctx context.Context, key string, value V, ttl time.Duration) error {
	for _, cache := range tc.caches {
		if err := cache.Set(ctx, key, value, ttl); err != nil {
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
	for _, cache := range tc.caches {
		if err := cache.Delete(ctx, key); err != nil && !errors.Is(err, ErrCacheMiss) {
			return err
		}
	}
	return nil
}
