# exp-go-cache

> **Note**: This is an experimental project for exploring multi-tier caching strategies and design patterns in Go. The API may change and is not recommended for production use.

A flexible, type-safe multi-tier caching library for Go with support for local and remote cache backends.

## Features

- **Type-Safe Generics**: Fully generic implementation using Go 1.18+ generics for compile-time type safety
- **Multi-Tier Caching**: Implements a tiered caching strategy (L1: Local Cache → L2: Remote Cache)
- **Two Caching Patterns**:
  - **TieredCache**: Single-key operations with singleflight protection
  - **BatchTieredCache**: Multi-key batch operations with optimized pipeline support
- **Pluggable Backends**: Support for multiple cache implementations
  - Local: [Ristretto](https://github.com/dgraph-io/ristretto) (high-performance in-memory cache)
  - Remote: Redis via [go-redis](https://github.com/redis/go-redis)
- **Flexible Serialization**: Multiple encoding formats
  - JSON (default)
  - MessagePack for better performance and smaller payload size
- **Compute Function**: Built-in support for cache-aside pattern with compute functions
- **Cache Stampede Protection**: TieredCacher uses singleflight to prevent duplicate compute function executions
- **Batch Optimization**: BatchTieredCacher uses Redis Pipeline for efficient multi-key operations
- **Context Support**: Full context.Context support for cancellation and timeouts

## Installation

```bash
go get github.com/naoto0822/exp-go-cache
```

## Quick Start

### Single-Key Operations with TieredCache

See [examples/tiered_cache.go](examples/tiered_cache.go) for a complete example.

### Multi-Key Operations with BatchTieredCache

See [examples/batch_tiered_cache.go](examples/batch_tiered_cache.go) for a complete example.

## Caching Strategies

### TieredCache - Single-Key Operations

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

### BatchTieredCache - Multi-Key Operations

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

- **TieredCache**: Single key lookups, need stampede protection, high concurrency for same key
- **BatchTieredCache**: Multiple keys at once, batch database support, minimize network overhead

## Performance Considerations

- **Ristretto** uses approximate algorithms (TinyLFU) for admission and eviction, providing excellent hit ratios
- **Redis Pipeline** batches multiple commands into single network round-trip (used in BatchTieredCache)
- **MessagePack** encoding is faster and produces smaller payloads compared to JSON
- **Singleflight** prevents thundering herd problem in TieredCache by deduplicating concurrent requests for the same key
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
