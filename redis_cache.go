package cacher

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache wraps go-redis client to implement the RemoteCacher interface with generic type support
type RedisCache[V any] struct {
	client *redis.Client
	coder  Coder[V]
}

// RedisCacheConfig holds configuration for RedisCache
type RedisCacheConfig struct {
	// Addr is the Redis server address (e.g., "localhost:6379")
	Addr string

	// Password for Redis authentication (optional)
	Password string

	// DB is the Redis database number (0-15, default is 0)
	DB int

	// DialTimeout is the timeout for establishing new connections
	DialTimeout time.Duration

	// ReadTimeout is the timeout for socket reads
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for socket writes
	WriteTimeout time.Duration

	// PoolSize is the maximum number of socket connections
	PoolSize int

	// MinIdleConns is the minimum number of idle connections
	MinIdleConns int
}

// DefaultRedisCacheConfig returns a default configuration
func DefaultRedisCacheConfig() *RedisCacheConfig {
	return &RedisCacheConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	}
}

// NewRedisCache creates a new RedisCache instance
func NewRedisCache[V any](config *RedisCacheConfig, coder Coder[V]) (*RedisCache[V], error) {
	if config == nil {
		config = DefaultRedisCacheConfig()
	}
	if coder == nil {
		coder = NewJSONCoder[V]()
	}
	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisCache[V]{
		client: client,
		coder:  coder,
	}, nil
}

// Get retrieves a value from Redis
func (r *RedisCache[V]) Get(ctx context.Context, key string) (V, error) {
	var zero V

	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return zero, ErrCacheMiss
		}
		return zero, err
	}

	// Decode using the configured coder
	value, err := r.coder.Decode([]byte(result))
	if err != nil {
		return zero, err
	}

	return value, nil
}

// Set stores a value in Redis with a TTL
func (r *RedisCache[V]) Set(ctx context.Context, key string, value V, ttl time.Duration) error {
	// Encode using the configured coder
	data, err := r.coder.Encode(value)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, ttl).Err()
}

// Delete removes a value from Redis
func (r *RedisCache[V]) Delete(ctx context.Context, key string) error {
	result, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return err
	}

	// If no keys were deleted, return ErrCacheMiss
	if result == 0 {
		return ErrCacheMiss
	}

	return nil
}

// BatchGet retrieves multiple values from Redis using Pipeline
// Returns a map of key-value pairs for found keys
// Missing keys are simply not included in the returned map
func (r *RedisCache[V]) BatchGet(ctx context.Context, keys []string) (map[string]V, error) {
	if len(keys) == 0 {
		return make(map[string]V), nil
	}

	// Use Pipeline for efficient batch operations
	pipe := r.client.Pipeline()

	// Queue all GET commands
	cmds := make([]*redis.StringCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		// Ignore redis.Nil errors as they indicate cache misses
		// Only return actual errors
	}

	// Collect results
	results := make(map[string]V, len(keys))
	for i, cmd := range cmds {
		result, err := cmd.Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				// Cache miss - skip this key
				continue
			}
			// Other errors - skip this key but continue processing
			continue
		}

		// Decode the value
		value, err := r.coder.Decode([]byte(result))
		if err != nil {
			// Decode error - skip this key
			continue
		}

		results[keys[i]] = value
	}

	return results, nil
}

// BatchSet stores multiple values in Redis with a TTL using Pipeline
// All items share the same TTL
func (r *RedisCache[V]) BatchSet(ctx context.Context, items map[string]V, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// Use Pipeline for efficient batch operations
	pipe := r.client.Pipeline()

	// Queue all SET commands
	for key, value := range items {
		// Encode the value
		data, err := r.coder.Encode(value)
		if err != nil {
			return err
		}
		pipe.Set(ctx, key, data, ttl)
	}

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	return err
}

// Close closes the Redis connection
func (r *RedisCache[V]) Close() error {
	return r.client.Close()
}

// Ping checks if the Redis server is reachable
func (r *RedisCache[V]) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
