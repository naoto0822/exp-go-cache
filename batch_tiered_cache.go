package cache

import (
	"context"
	"time"
)

// BatchComputeFunc is a function that computes multiple values when cache misses occur
// It receives a slice of keys and returns a map of key-value pairs
type BatchComputeFunc[V any] func(ctx context.Context, keys []string) (map[string]V, error)

// BatchTieredCache implements multi-key cache operations with tiered caching strategy
// Strategy: caches[0] (L1) → caches[1] (L2) → ... → caches[n] (Ln)
// Optimized for batch operations where the compute function can fetch multiple keys efficiently
type BatchTieredCache[V any] struct {
	caches []BatchCacher[V]
}

// NewBatchTieredCache creates a new batch tiered cache with dependency injection
// caches is a slice where caches[0] is L1 (fastest), caches[1] is L2, etc.
// Empty or nil caches in the slice are skipped
func NewBatchTieredCache[V any](caches ...BatchCacher[V]) *BatchTieredCache[V] {
	// Filter out nil caches
	validCaches := make([]BatchCacher[V], 0, len(caches))
	for _, cache := range caches {
		if cache != nil {
			validCaches = append(validCaches, cache)
		}
	}
	return &BatchTieredCache[V]{
		caches: validCaches,
	}
}

// BatchGet retrieves multiple values using the tiered caching strategy:
// 1. Check L1, L2, ..., Ln in order using BatchGet
// 2. For each tier hit, populate upper tiers
// 3. For all misses, execute batchComputeFn to fetch all at once
// 4. Populate all tiers with computed values
// Returns a map of successfully retrieved values (key -> value)
func (bc *BatchTieredCache[V]) BatchGet(ctx context.Context, keys []string, ttl time.Duration, batchComputeFn BatchComputeFunc[V]) (map[string]V, error) {
	if len(keys) == 0 {
		return make(map[string]V), nil
	}

	results := make(map[string]V)
	remainingKeys := keys

	// Try each cache tier in order
	for tierIndex, cache := range bc.caches {
		if len(remainingKeys) == 0 {
			break
		}

		tierResults, err := cache.BatchGet(ctx, remainingKeys)
		if err == nil && len(tierResults) > 0 {
			// Add tier hits to results
			for k, v := range tierResults {
				results[k] = v
			}

			// Populate upper tiers if this is L2 or below
			if tierIndex > 0 {
				_ = bc.populateUpperTiers(ctx, tierResults, ttl, tierIndex)
			}

			// Update remaining keys (tier misses)
			remainingKeys = filterMissingKeys(remainingKeys, tierResults)
		}
	}

	// If all keys were found in cache, return early
	if len(remainingKeys) == 0 {
		return results, nil
	}

	// Execute batch compute for remaining keys
	computedValues, err := batchComputeFn(ctx, remainingKeys)
	if err != nil {
		return results, err
	}

	// Populate all caches with computed values
	if len(computedValues) > 0 {
		for _, cache := range bc.caches {
			_ = cache.BatchSet(ctx, computedValues, ttl)
		}

		// Add computed values to results
		for k, v := range computedValues {
			results[k] = v
		}
	}

	return results, nil
}

// populateUpperTiers writes values to all cache tiers above the specified tier
func (bc *BatchTieredCache[V]) populateUpperTiers(ctx context.Context, items map[string]V, ttl time.Duration, foundTierIndex int) error {
	for i := 0; i < foundTierIndex && i < len(bc.caches); i++ {
		if err := bc.caches[i].BatchSet(ctx, items, ttl); err != nil {
			return err
		}
	}
	return nil
}

// BatchSet stores multiple values in all cache tiers
// All items share the same TTL
func (bc *BatchTieredCache[V]) BatchSet(ctx context.Context, items map[string]V, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	for _, cache := range bc.caches {
		if err := cache.BatchSet(ctx, items, ttl); err != nil {
			return err
		}
	}

	return nil
}

// filterMissingKeys returns keys that are not present in the foundKeys map
func filterMissingKeys[V any](keys []string, foundKeys map[string]V) []string {
	missing := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, found := foundKeys[key]; !found {
			missing = append(missing, key)
		}
	}
	return missing
}
