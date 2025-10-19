package cacher

import (
	"context"
	"time"
)

// BatchComputeFunc is a function that computes multiple values when cache misses occur
// It receives a slice of keys and returns a map of key-value pairs
type BatchComputeFunc[V any] func(ctx context.Context, keys []string) (map[string]V, error)

// BatchTieredCacher implements multi-key cache operations with tiered caching strategy
// Strategy: L1 (Local Cache) → L2 (Remote Cache)
// Optimized for batch operations where the compute function can fetch multiple keys efficiently
type BatchTieredCacher[V any] struct {
	localCache  BatchLocalCacher[V]
	remoteCache BatchRemoteCacher[V]
}

// NewBatchTieredCacher creates a new batch tiered cacher with dependency injection
// Both localCache and remoteCache are optional (can be nil)
func NewBatchTieredCacher[V any](localCache BatchLocalCacher[V], remoteCache BatchRemoteCacher[V]) *BatchTieredCacher[V] {
	return &BatchTieredCacher[V]{
		localCache:  localCache,
		remoteCache: remoteCache,
	}
}

// BatchGet retrieves multiple values using the tiered caching strategy:
// 1. Check L1 (local cache) using BatchGet
// 2. For L1 misses, check L2 (remote cache) using BatchGet and populate L1
// 3. For L2 misses, execute batchComputeFn to fetch all at once
// 4. Populate both L1 and L2 with computed values
// Returns a map of successfully retrieved values (key -> value)
func (bc *BatchTieredCacher[V]) BatchGet(ctx context.Context, keys []string, ttl time.Duration, batchComputeFn BatchComputeFunc[V]) (map[string]V, error) {
	if len(keys) == 0 {
		return make(map[string]V), nil
	}

	results := make(map[string]V)
	remainingKeys := keys

	// Step 1: Try to get from L1 (local cache)
	if bc.localCache != nil {
		l1Results, err := bc.localCache.BatchGet(ctx, remainingKeys)
		if err == nil && len(l1Results) > 0 {
			// Add L1 hits to results
			for k, v := range l1Results {
				results[k] = v
			}

			// Update remaining keys (L1 misses)
			remainingKeys = filterMissingKeys(remainingKeys, l1Results)
		}
	}

	// If all keys were found in L1, return early
	if len(remainingKeys) == 0 {
		return results, nil
	}

	// Step 2: Try to get from L2 (remote cache)
	if bc.remoteCache != nil {
		l2Results, err := bc.remoteCache.BatchGet(ctx, remainingKeys)
		if err == nil && len(l2Results) > 0 {
			// Add L2 hits to results
			for k, v := range l2Results {
				results[k] = v
			}

			// Populate L1 with L2 hits
			if bc.localCache != nil {
				_ = bc.localCache.BatchSet(ctx, l2Results, ttl)
			}

			// Update remaining keys (L2 misses)
			remainingKeys = filterMissingKeys(remainingKeys, l2Results)
		}
	}

	// If all keys were found in cache, return early
	if len(remainingKeys) == 0 {
		return results, nil
	}

	// Step 3: Execute batch compute for remaining keys
	computedValues, err := batchComputeFn(ctx, remainingKeys)
	if err != nil {
		return results, err
	}

	// Step 4: Populate caches with computed values
	if len(computedValues) > 0 {
		// Set in L1
		if bc.localCache != nil {
			_ = bc.localCache.BatchSet(ctx, computedValues, ttl)
		}

		// Set in L2
		if bc.remoteCache != nil {
			_ = bc.remoteCache.BatchSet(ctx, computedValues, ttl)
		}

		// Add computed values to results
		for k, v := range computedValues {
			results[k] = v
		}
	}

	return results, nil
}

// BatchSet stores multiple values in all cache tiers
// All items share the same TTL
func (bc *BatchTieredCacher[V]) BatchSet(ctx context.Context, items map[string]V, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// Set in L1
	if bc.localCache != nil {
		if err := bc.localCache.BatchSet(ctx, items, ttl); err != nil {
			return err
		}
	}

	// Set in L2
	if bc.remoteCache != nil {
		if err := bc.remoteCache.BatchSet(ctx, items, ttl); err != nil {
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
