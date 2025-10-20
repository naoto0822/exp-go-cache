package examples

import (
	"context"
	"fmt"
	"time"

	cache "github.com/naoto0822/exp-go-cache"
)

type Book struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func exampleBatchTieredCache() {
	ctx := context.Background()

	// Setup caches (same as TieredCache)
	localCache, _ := cache.NewRistrettoCache[Book](nil)
	defer localCache.Close()

	remoteCache, _ := cache.NewRedisCache[Book](cache.DefaultRedisCacheConfig(), nil)
	defer remoteCache.Close()

	// Create batch tiered cache
	batchCache := cache.NewBatchTieredCache(localCache, remoteCache)

	// Batch get with compute function
	keys := []string{"book:1", "book:2", "book:3"}
	books, err := batchCache.BatchGet(ctx, keys, 5*time.Minute, func(ctx context.Context, missedKeys []string) (map[string]Book, error) {
		// This function receives only cache-missed keys
		fmt.Printf("Cache miss for keys: %v - fetching from database\n", missedKeys)

		// Fetch all books from database in one query (efficient!)
		results := make(map[string]Book)
		for _, key := range missedKeys {
			// Simulate batch database query
			results[key] = Book{ID: 1, Name: "Alice"}
		}
		return results, nil
	})

	if err != nil {
		panic(err)
	}

	fmt.Printf("Books: %+v\n", books)
}
