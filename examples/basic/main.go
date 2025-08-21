package main

import (
	"fmt"
	"log"
	"time"

	"github.com/nayan9229/fastcache"
)

func main() {
	fmt.Println("=== FastCache Basic Usage Example ===")

	// Create cache with default configuration
	cache := fastcache.New(fastcache.DefaultConfig())
	defer cache.Close()

	// Basic Set/Get operations
	basicOperations(cache)

	// TTL operations
	ttlOperations(cache)

	// Different value types
	differentTypes(cache)

	// Statistics
	showStatistics(cache)
}

func basicOperations(cache *fastcache.Cache) {
	fmt.Println("1. Basic Set/Get Operations:")

	// Set some values
	err := cache.Set("user:123", map[string]interface{}{
		"id":    123,
		"name":  "John Doe",
		"email": "john@example.com",
		"role":  "admin",
	})
	if err != nil {
		log.Fatal(err)
	}

	err = cache.Set("product:456", map[string]interface{}{
		"id":    456,
		"name":  "Laptop",
		"price": 999.99,
		"stock": 50,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Get values
	if user, exists := cache.Get("user:123"); exists {
		fmt.Printf("Found user: %+v\n", user)
	} else {
		fmt.Println("User not found")
	}

	if product, exists := cache.Get("product:456"); exists {
		fmt.Printf("Found product: %+v\n", product)
	} else {
		fmt.Println("Product not found")
	}

	// Try to get non-existent key
	if _, exists := cache.Get("user:999"); !exists {
		fmt.Println("User 999 not found (as expected)")
	}

	fmt.Println()
}

func ttlOperations(cache *fastcache.Cache) {
	fmt.Println("2. TTL (Time-To-Live) Operations:")

	// Set with custom TTL
	err := cache.Set("session:abc123", "user-session-data", 3*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	err = cache.Set("temp:data", "temporary-value", 1*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	// Check immediately
	if value, exists := cache.Get("session:abc123"); exists {
		fmt.Printf("Session found: %s\n", value)
	}

	if value, exists := cache.Get("temp:data"); exists {
		fmt.Printf("Temp data found: %s\n", value)
	}

	// Wait for temp data to expire
	fmt.Println("Waiting 2 seconds for temp data to expire...")
	time.Sleep(2 * time.Second)

	// Check again
	if _, exists := cache.Get("temp:data"); !exists {
		fmt.Println("Temp data expired (as expected)")
	}

	if value, exists := cache.Get("session:abc123"); exists {
		fmt.Printf("Session still exists: %s\n", value)
	}

	// Wait for session to expire
	fmt.Println("Waiting 2 more seconds for session to expire...")
	time.Sleep(2 * time.Second)

	if _, exists := cache.Get("session:abc123"); !exists {
		fmt.Println("Session expired (as expected)")
	}

	fmt.Println()
}

func differentTypes(cache *fastcache.Cache) {
	fmt.Println("3. Different Value Types:")

	// String
	cache.Set("string_key", "Hello, World!")

	// Integer
	cache.Set("int_key", 42)

	// Float
	cache.Set("float_key", 3.14159)

	// Boolean
	cache.Set("bool_key", true)

	// Byte slice
	cache.Set("bytes_key", []byte("binary data"))

	// Slice
	cache.Set("slice_key", []string{"apple", "banana", "cherry"})

	// Map
	cache.Set("map_key", map[string]int{
		"red":   1,
		"green": 2,
		"blue":  3,
	})

	// Struct
	type Person struct {
		Name string
		Age  int
		City string
	}
	cache.Set("struct_key", Person{
		Name: "Alice",
		Age:  30,
		City: "New York",
	})

	// Retrieve and display
	keys := []string{"string_key", "int_key", "float_key", "bool_key",
		"bytes_key", "slice_key", "map_key", "struct_key"}

	for _, key := range keys {
		if value, exists := cache.Get(key); exists {
			fmt.Printf("%s: %v (%T)\n", key, value, value)
		}
	}

	fmt.Println()
}

func showStatistics(cache *fastcache.Cache) {
	fmt.Println("4. Cache Statistics:")

	// Perform some operations to generate stats
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("stats_key_%d", i)
		value := fmt.Sprintf("stats_value_%d", i)
		cache.Set(key, value)
	}

	// Generate some hits and misses
	for i := 0; i < 50; i++ {
		cache.Get(fmt.Sprintf("stats_key_%d", i)) // hits
	}

	for i := 200; i < 250; i++ {
		cache.Get(fmt.Sprintf("missing_key_%d", i)) // misses
	}

	// Get comprehensive stats
	stats := cache.GetStats()
	fmt.Printf("Cache Statistics:\n")
	fmt.Printf("- Total Entries: %d\n", stats.TotalEntries)
	fmt.Printf("- Memory Usage: %s (%.1f%%)\n", stats.MemoryUsage, stats.MemoryPercent)
	fmt.Printf("- Hit Count: %d\n", stats.HitCount)
	fmt.Printf("- Miss Count: %d\n", stats.MissCount)
	fmt.Printf("- Hit Ratio: %.2f%%\n", stats.HitRatio*100)
	fmt.Printf("- Shard Count: %d\n", stats.ShardCount)

	// Get memory info
	memInfo := cache.GetMemoryInfo()
	fmt.Printf("\nMemory Information:\n")
	fmt.Printf("- Used: %s\n", memInfo.UsedFormatted)
	fmt.Printf("- Available: %s\n", memInfo.AvailableFormatted)
	fmt.Printf("- Maximum: %s\n", memInfo.MaxFormatted)
	fmt.Printf("- Usage: %.1f%%\n", memInfo.Percent)

	// Get performance metrics
	perfMetrics := cache.GetPerformanceMetrics()
	fmt.Printf("\nPerformance Metrics:\n")
	fmt.Printf("- Total Operations: %d\n", perfMetrics.TotalOperations)
	fmt.Printf("- Hit Rate: %.2f%%\n", perfMetrics.HitRate*100)
	fmt.Printf("- Miss Rate: %.2f%%\n", perfMetrics.MissRate*100)
	fmt.Printf("- Average Shard Load: %.1f\n", perfMetrics.AvgShardLoad)
	fmt.Printf("- Max Shard Load: %d\n", perfMetrics.MaxShardLoad)
	fmt.Printf("- Min Shard Load: %d\n", perfMetrics.MinShardLoad)
	fmt.Printf("- Load Balance Variance: %.2f\n", perfMetrics.LoadBalance)

	fmt.Println()
}
