# exp-go-cacher

> **Note**: This is an experimental project for exploring multi-tier caching strategies and design patterns in Go. The API may change and is not recommended for production use.

A flexible, type-safe multi-tier caching library for Go with support for local and remote cache backends.

## Features

- **Type-Safe Generics**: Fully generic implementation using Go 1.18+ generics for compile-time type safety
- **Multi-Tier Caching**: Implements a tiered caching strategy (L1: Local Cache → L2: Remote Cache)
- **Two Caching Patterns**:
  - **TieredCacher**: Single-key operations with singleflight protection
  - **BatchCacher**: Multi-key batch operations with optimized pipeline support
- **Pluggable Backends**: Support for multiple cache implementations
  - Local: [Ristretto](https://github.com/dgraph-io/ristretto) (high-performance in-memory cache)
  - Remote: Redis via [go-redis](https://github.com/redis/go-redis)
- **Flexible Serialization**: Multiple encoding formats
  - JSON (default)
  - MessagePack for better performance and smaller payload size
- **Compute Function**: Built-in support for cache-aside pattern with compute functions
- **Cache Stampede Protection**: TieredCacher uses singleflight to prevent duplicate compute function executions
- **Batch Optimization**: BatchCacher uses Redis Pipeline for efficient multi-key operations
- **Context Support**: Full context.Context support for cancellation and timeouts

## Installation

```bash
go get github.com/naoto0822/exp-go-cacher
```

## Quick Start

### Single-Key Operations with TieredCacher

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/naoto0822/exp-go-cacher"
)

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func main() {
    ctx := context.Background()

    // Setup local cache (Ristretto)
    localCache, err := cacher.NewRistrettoCache[User](nil)
    if err != nil {
        panic(err)
    }
    defer localCache.Close()

    // Setup remote cache (Redis)
    redisConfig := cacher.DefaultRedisCacheConfig()
    remoteCache, err := cacher.NewRedisCache[User](redisConfig, nil)
    if err != nil {
        panic(err)
    }
    defer remoteCache.Close()

    // Create tiered cacher
    tieredCacher := cacher.NewTieredCacher[User](localCache, remoteCache)

    // Use with compute function (cache-aside pattern)
    user, err := tieredCacher.Get(ctx, "user:123", 5*time.Minute, func(ctx context.Context, key string) (User, error) {
        // This function is called only on cache miss
        fmt.Println("Cache miss - fetching from database")
        return User{ID: 123, Name: "Alice"}, nil
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("User: %+v\n", user)
}
```

### Multi-Key Operations with BatchCacher

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/naoto0822/exp-go-cacher"
)

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func main() {
    ctx := context.Background()

    // Setup caches (same as TieredCacher)
    localCache, _ := cacher.NewRistrettoCache[User](nil)
    defer localCache.Close()

    remoteCache, _ := cacher.NewRedisCache[User](cacher.DefaultRedisCacheConfig(), nil)
    defer remoteCache.Close()

    // Create batch cacher
    batchCacher := cacher.NewBatchCacher[User](localCache, remoteCache)

    // Batch get with compute function
    keys := []string{"user:1", "user:2", "user:3"}
    users, err := batchCacher.BatchGet(ctx, keys, 5*time.Minute, func(ctx context.Context, missedKeys []string) (map[string]User, error) {
        // This function receives only cache-missed keys
        fmt.Printf("Cache miss for keys: %v - fetching from database\n", missedKeys)

        // Fetch all users from database in one query (efficient!)
        results := make(map[string]User)
        for _, key := range missedKeys {
            // Simulate batch database query
            results[key] = User{ID: 1, Name: "Alice"}
        }
        return results, nil
    })

    if err != nil {
        panic(err)
    }

    fmt.Printf("Users: %+v\n", users)
}
```

## Caching Strategies

### TieredCacher - Single-Key Operations

Optimized for single-key lookups with cache stampede protection.

**Cache Flow:**
```
Get Request (single key)
    ↓
Check L1 (Local) ──Hit──→ Return Value
    ↓ Miss
Check L2 (Remote) ──Hit──→ Populate L1 → Return Value
    ↓ Miss
Execute Compute Function (via singleflight) → Populate L1 & L2 → Return Value
```

**Cache Stampede Protection:**

Uses [singleflight](https://pkg.go.dev/golang.org/x/sync/singleflight) to ensure only one compute function executes per key:

- **Without singleflight**: 100 concurrent requests for same key = 100 database calls
- **With singleflight**: 100 concurrent requests for same key = 1 database call (others wait and share result)

### BatchCacher - Multi-Key Operations

Optimized for multi-key batch operations with efficient pipeline support.

**Cache Flow:**
```
BatchGet Request (multiple keys)
    ↓
BatchGet L1 (Local) ──Hits──→ Add to results
    ↓ Misses
BatchGet L2 (Remote via Pipeline) ──Hits──→ Populate L1 + Add to results
    ↓ Misses
Execute Batch Compute Function → Populate L1 & L2 → Add to results
    ↓
Return all results
```

**Batch Optimization:**

- **L1 (Ristretto)**: Simple loop (fast memory access)
- **L2 (Redis)**: Uses Pipeline for 1 network round-trip instead of N
- **Compute Function**: Client can implement batch database query (N queries → 1 query)

**Performance Comparison:**

| Operation | Without Batch | With Batch |
|-----------|--------------|------------|
| 100 keys Redis fetch | 100 network calls | 1 network call (Pipeline) |
| 100 keys DB fetch | 100 SQL queries | 1 SQL query (IN clause) |

**When to Use:**

- **TieredCacher**: Single key lookups, need stampede protection, high concurrency for same key
- **BatchCacher**: Multiple keys at once, batch database support, minimize network overhead

## Architecture

### Cache Tiers

1. **L1 (Local Cache)**: Fast in-memory cache using Ristretto
2. **L2 (Remote Cache)**: Distributed cache using Redis

### Interfaces

```go
// Single-key interfaces
type LocalCacher[V any] interface {
    Get(ctx context.Context, key string) (V, error)
    Set(ctx context.Context, key string, value V, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}

type RemoteCacher[V any] interface {
    Get(ctx context.Context, key string) (V, error)
    Set(ctx context.Context, key string, value V, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}

// Batch interfaces (extends single-key interfaces)
type BatchLocalCacher[V any] interface {
    LocalCacher[V]
    BatchGet(ctx context.Context, keys []string) (map[string]V, error)
    BatchSet(ctx context.Context, items map[string]V, ttl time.Duration) error
}

type BatchRemoteCacher[V any] interface {
    RemoteCacher[V]
    BatchGet(ctx context.Context, keys []string) (map[string]V, error)
    BatchSet(ctx context.Context, items map[string]V, ttl time.Duration) error
}
```

## Performance Considerations

- **Ristretto** uses approximate algorithms (TinyLFU) for admission and eviction, providing excellent hit ratios
- **Redis Pipeline** batches multiple commands into single network round-trip (used in BatchCacher)
- **MessagePack** encoding is faster and produces smaller payloads compared to JSON
- **Singleflight** prevents thundering herd problem in TieredCacher by deduplicating concurrent requests for the same key
- **Batch Operations** minimize N+1 query problems by fetching multiple keys efficiently
- Both cache tiers are populated on compute to maximize cache hits
- Context cancellation is respected throughout the caching flow

## Dependencies

- [github.com/dgraph-io/ristretto](https://github.com/dgraph-io/ristretto) - High-performance in-memory cache
- [github.com/redis/go-redis/v9](https://github.com/redis/go-redis) - Redis client for Go
- [github.com/hashicorp/go-msgpack/v2](https://github.com/hashicorp/go-msgpack) - MessagePack encoding
- [golang.org/x/sync/singleflight](https://pkg.go.dev/golang.org/x/sync/singleflight) - Cache stampede protection

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
