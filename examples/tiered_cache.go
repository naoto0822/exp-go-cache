package examples

import (
	"context"
	"fmt"
	"time"

	cache "github.com/naoto0822/exp-go-cache"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func exampleTieredCache() {
	ctx := context.Background()

	// Setup local cache (Ristretto)
	localCache, err := cache.NewRistrettoCache[User](nil)
	if err != nil {
		panic(err)
	}
	defer localCache.Close()

	// Setup remote cache (Redis)
	redisConfig := cache.DefaultRedisCacheConfig()
	remoteCache, err := cache.NewRedisCache[User](redisConfig, nil)
	if err != nil {
		panic(err)
	}
	defer remoteCache.Close()

	// Create tiered cache
	tieredCache := cache.NewTieredCache(localCache, remoteCache)

	// Use with compute function (cache-aside pattern)
	user, err := tieredCache.Get(ctx, "user:123", 5*time.Minute, func(ctx context.Context, key string) (User, error) {
		// This function is called only on cache miss
		fmt.Println("Cache miss - fetching from database")
		return User{ID: 123, Name: "Alice"}, nil
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("User: %+v\n", user)
}
