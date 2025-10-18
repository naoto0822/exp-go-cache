# exp-go-cacher

A flexible, type-safe multi-tier caching library for Go with support for local and remote cache backends.

## Features

- **Type-Safe Generics**: Fully generic implementation using Go 1.18+ generics for compile-time type safety
- **Multi-Tier Caching**: Implements a tiered caching strategy (L1: Local Cache → L2: Remote Cache)
- **Pluggable Backends**: Support for multiple cache implementations
  - Local: [Ristretto](https://github.com/dgraph-io/ristretto) (high-performance in-memory cache)
  - Remote: Redis via [go-redis](https://github.com/redis/go-redis)
- **Flexible Serialization**: Multiple encoding formats
  - JSON (default)
  - MessagePack for better performance and smaller payload size
- **Compute Function**: Built-in support for cache-aside pattern with compute functions
- **Context Support**: Full context.Context support for cancellation and timeouts

## Installation

```bash
go get github.com/naoto0822/exp-go-cacher
```

## Quick Start

### Basic Usage with Tiered Cache

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

### Using Only Local Cache

```go
// Create local-only cache
localCache, _ := cacher.NewRistrettoCache[string](nil)
tieredCacher := cacher.NewTieredCacher[string](localCache, nil)

ctx := context.Background()
value, err := tieredCacher.Get(ctx, "key", time.Minute, func(ctx context.Context, key string) (string, error) {
    return "computed value", nil
})
```

### Using Only Remote Cache

```go
// Create remote-only cache
redisCache, _ := cacher.NewRedisCache[string](nil, nil)
tieredCacher := cacher.NewTieredCacher[string](nil, redisCache)

ctx := context.Background()
value, err := tieredCacher.Get(ctx, "key", time.Hour, func(ctx context.Context, key string) (string, error) {
    return "computed value", nil
})
```

## Configuration

### Ristretto Cache Configuration

```go
config := &cacher.RistrettoCacheConfig{
    NumCounters: 1e7,     // 10 million counters
    MaxCost:     1 << 30, // 1GB max cost
    BufferItems: 64,      // Buffer size
}

cache, err := cacher.NewRistrettoCache[MyType](config)
```

### Redis Cache Configuration

```go
config := &cacher.RedisCacheConfig{
    Addr:         "localhost:6379",
    Password:     "",
    DB:           0,
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
    PoolSize:     10,
    MinIdleConns: 2,
}

cache, err := cacher.NewRedisCache[MyType](config, cacher.NewJSONCoder[MyType]())
```

## Serialization

### Using JSON (Default)

```go
jsonCoder := cacher.NewJSONCoder[User]()
cache, err := cacher.NewRedisCache[User](config, jsonCoder)
```

### Using MessagePack

```go
msgpackCoder := cacher.NewMessagePackCoder[User]()
cache, err := cacher.NewRedisCache[User](config, msgpackCoder)
```

### Custom Coder

Implement the `Coder[V]` interface:

```go
type Coder[V any] interface {
    Encode(value V) ([]byte, error)
    Decode(data []byte) (V, error)
}
```

## API Reference

### TieredCacher

```go
type TieredCacher[V any] struct { ... }

// Get retrieves a value using tiered caching with compute function
func (tc *TieredCacher[V]) Get(ctx context.Context, key string, ttl time.Duration, computeFn ComputeFunc[V]) (V, error)

// Set stores a value in all cache tiers
func (tc *TieredCacher[V]) Set(ctx context.Context, key string, value V, ttl time.Duration) error

// Delete removes a key from all cache tiers
func (tc *TieredCacher[V]) Delete(ctx context.Context, key string) error
```

### Cache Interfaces

```go
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
```

## Caching Strategy

The `TieredCacher` implements a multi-tier caching strategy:

1. **L1 (Local Cache)**: Fast in-memory cache using Ristretto
2. **L2 (Remote Cache)**: Distributed cache using Redis
3. **Compute Function**: Fallback function to compute/fetch the value

### Cache Flow

```
Get Request
    ↓
Check L1 (Local) ──Hit──→ Return Value
    ↓ Miss
Check L2 (Remote) ──Hit──→ Populate L1 → Return Value
    ↓ Miss
Execute Compute Function → Populate L1 & L2 → Return Value
```

## Error Handling

The library uses a sentinel error for cache misses:

```go
if errors.Is(err, cacher.ErrCacheMiss) {
    // Handle cache miss
}
```

## Performance Considerations

- **Ristretto** uses approximate algorithms (TinyLFU) for admission and eviction, providing excellent hit ratios
- **MessagePack** encoding is faster and produces smaller payloads compared to JSON
- Both cache tiers are populated on compute to maximize cache hits
- Context cancellation is respected throughout the caching flow

## Dependencies

- [github.com/dgraph-io/ristretto](https://github.com/dgraph-io/ristretto) - High-performance in-memory cache
- [github.com/redis/go-redis/v9](https://github.com/redis/go-redis) - Redis client for Go
- [github.com/hashicorp/go-msgpack/v2](https://github.com/hashicorp/go-msgpack) - MessagePack encoding

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
