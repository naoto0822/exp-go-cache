# exp-go-cacher

> **Note**: This is an experimental project for exploring multi-tier caching strategies and design patterns in Go. The API may change and is not recommended for production use.

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
