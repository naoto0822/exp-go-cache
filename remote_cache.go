package memoizer

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps go-redis client to implement the RemoteCacher interface with generic type support
type RedisClient[V any] struct {
	client *redis.Client
	coder  Coder[V]
}

// RedisClientConfig holds configuration for RedisClient
type RedisClientConfig struct {
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

// DefaultRedisClientConfig returns a default configuration
func DefaultRedisClientConfig() *RedisClientConfig {
	return &RedisClientConfig{
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

// NewRedisClient creates a new RedisClient instance
func NewRedisClient[V any](config *RedisClientConfig, coder Coder[V]) (*RedisClient[V], error) {
	if config == nil {
		config = DefaultRedisClientConfig()
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

	return &RedisClient[V]{
		client: client,
		coder:  coder,
	}, nil
}

// Get retrieves a value from Redis
func (r *RedisClient[V]) Get(ctx context.Context, key string) (V, error) {
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
func (r *RedisClient[V]) Set(ctx context.Context, key string, value V, ttl time.Duration) error {
	// Encode using the configured coder
	data, err := r.coder.Encode(value)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, ttl).Err()
}

// Delete removes a value from Redis
func (r *RedisClient[V]) Delete(ctx context.Context, key string) error {
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

// Close closes the Redis connection
func (r *RedisClient[V]) Close() error {
	return r.client.Close()
}

// Ping checks if the Redis server is reachable
func (r *RedisClient[V]) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
