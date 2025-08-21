package fastcache

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func BenchmarkSet(b *testing.B) {
	cache := New(DefaultConfig())
	defer cache.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bench_key_%d", i)
			value := fmt.Sprintf("bench_value_%d", i)
			cache.Set(key, value)
			i++
		}
	})
}

func BenchmarkGet(b *testing.B) {
	cache := New(DefaultConfig())
	defer cache.Close()

	// Pre-populate with data
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("bench_key_%d", i)
		value := fmt.Sprintf("bench_value_%d", i)
		cache.Set(key, value)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := fmt.Sprintf("bench_key_%d", rand.Intn(10000))
			cache.Get(key)
		}
	})
}

func BenchmarkGetMiss(b *testing.B) {
	cache := New(DefaultConfig())
	defer cache.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("missing_key_%d", i)
			cache.Get(key)
			i++
		}
	})
}

func BenchmarkDelete(b *testing.B) {
	cache := New(DefaultConfig())
	defer cache.Close()

	// Pre-populate with data
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("delete_key_%d", i)
		cache.Set(key, fmt.Sprintf("value_%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("delete_key_%d", i)
		cache.Delete(key)
	}
}

func BenchmarkMixed(b *testing.B) {
	cache := New(DefaultConfig())
	defer cache.Close()

	// Pre-populate with some data
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("mixed_key_%d", i)
		value := fmt.Sprintf("mixed_value_%d", i)
		cache.Set(key, value)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 1000
		for pb.Next() {
			if rand.Float32() < 0.3 { // 30% writes
				key := fmt.Sprintf("mixed_key_%d", i)
				value := fmt.Sprintf("mixed_value_%d", i)
				cache.Set(key, value)
				i++
			} else { // 70% reads
				key := fmt.Sprintf("mixed_key_%d", rand.Intn(i))
				cache.Get(key)
			}
		}
	})
}

func BenchmarkSetWithTTL(b *testing.B) {
	cache := New(DefaultConfig())
	defer cache.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("ttl_key_%d", i)
			value := fmt.Sprintf("ttl_value_%d", i)
			cache.Set(key, value, 5*time.Minute)
			i++
		}
	})
}

func BenchmarkGetStats(b *testing.B) {
	cache := New(DefaultConfig())
	defer cache.Close()

	// Add some data
	for i := 0; i < 1000; i++ {
		cache.Set(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetStats()
	}
}

func BenchmarkHighConcurrency(b *testing.B) {
	cache := New(HighConcurrencyConfig())
	defer cache.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if rand.Float32() < 0.5 {
				key := fmt.Sprintf("concurrent_key_%d", i)
				value := fmt.Sprintf("concurrent_value_%d", i)
				cache.Set(key, value)
			} else {
				key := fmt.Sprintf("concurrent_key_%d", rand.Intn(i+1))
				cache.Get(key)
			}
			i++
		}
	})
}

func BenchmarkLargeValues(b *testing.B) {
	cache := New(DefaultConfig())
	defer cache.Close()

	// Create 1KB value
	largeValue := make([]byte, 1024)
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("large_key_%d", i)
			cache.Set(key, largeValue)
			i++
		}
	})
}

func BenchmarkSmallValues(b *testing.B) {
	cache := New(DefaultConfig())
	defer cache.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("small_key_%d", i)
			cache.Set(key, "x") // Single character
			i++
		}
	})
}

// Benchmark different shard counts
func BenchmarkShardCount64(b *testing.B) {
	benchmarkWithShardCount(b, 64)
}

func BenchmarkShardCount256(b *testing.B) {
	benchmarkWithShardCount(b, 256)
}

func BenchmarkShardCount1024(b *testing.B) {
	benchmarkWithShardCount(b, 1024)
}

func BenchmarkShardCount4096(b *testing.B) {
	benchmarkWithShardCount(b, 4096)
}

func benchmarkWithShardCount(b *testing.B, shardCount int) {
	config := &Config{
		MaxMemoryBytes:  512 * 1024 * 1024,
		ShardCount:      shardCount,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Minute,
	}

	cache := New(config)
	defer cache.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if rand.Float32() < 0.3 {
				key := fmt.Sprintf("shard_key_%d", i)
				value := fmt.Sprintf("shard_value_%d", i)
				cache.Set(key, value)
			} else {
				key := fmt.Sprintf("shard_key_%d", rand.Intn(i+1))
				cache.Get(key)
			}
			i++
		}
	})
}

// Memory usage benchmarks
func BenchmarkMemoryUsage(b *testing.B) {
	cache := New(DefaultConfig())
	defer cache.Close()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("memory_key_%d", i)
		value := fmt.Sprintf("memory_value_%d", i)
		cache.Set(key, value)
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocatedMB := float64(m2.Alloc-m1.Alloc) / 1024 / 1024
	b.ReportMetric(allocatedMB, "MB")
}

// Benchmark eviction performance
func BenchmarkEviction(b *testing.B) {
	config := &Config{
		MaxMemoryBytes:  1024 * 1024, // 1MB limit
		ShardCount:      64,
		DefaultTTL:      0,
		CleanupInterval: time.Second,
	}

	cache := New(config)
	defer cache.Close()

	// Fill cache to trigger eviction
	largeValue := make([]byte, 512) // 512 bytes per entry
	for i := 0; i < 3000; i++ {     // Try to add ~1.5MB
		key := fmt.Sprintf("evict_key_%d", i)
		cache.Set(key, largeValue)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("new_key_%d", i)
		cache.Set(key, largeValue) // This should trigger eviction
	}
}

// Comprehensive performance test
func BenchmarkComprehensive(b *testing.B) {
	cache := New(DefaultConfig())
	defer cache.Close()

	// Pre-populate
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("comp_key_%d", i)
		value := fmt.Sprintf("comp_value_%d", i)
		cache.Set(key, value)
	}

	operations := []func(int){
		func(i int) { // SET
			key := fmt.Sprintf("comp_key_%d", i)
			value := fmt.Sprintf("comp_value_%d", i)
			cache.Set(key, value)
		},
		func(i int) { // GET hit
			key := fmt.Sprintf("comp_key_%d", rand.Intn(10000))
			cache.Get(key)
		},
		func(i int) { // GET miss
			key := fmt.Sprintf("miss_key_%d", i)
			cache.Get(key)
		},
		func(i int) { // DELETE
			key := fmt.Sprintf("comp_key_%d", rand.Intn(10000))
			cache.Delete(key)
		},
		func(i int) { // SET with TTL
			key := fmt.Sprintf("ttl_key_%d", i)
			value := fmt.Sprintf("ttl_value_%d", i)
			cache.Set(key, value, 5*time.Minute)
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 10000
		for pb.Next() {
			op := operations[rand.Intn(len(operations))]
			op(i)
			i++
		}
	})
}

// QPS measurement benchmark
func BenchmarkQPS(b *testing.B) {
	cache := New(DefaultConfig())
	defer cache.Close()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("qps_key_%d", i)
		value := fmt.Sprintf("qps_value_%d", i)
		cache.Set(key, value)
	}

	start := time.Now()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 1000
		for pb.Next() {
			if rand.Float32() < 0.3 {
				key := fmt.Sprintf("qps_key_%d", i)
				value := fmt.Sprintf("qps_value_%d", i)
				cache.Set(key, value)
				i++
			} else {
				key := fmt.Sprintf("qps_key_%d", rand.Intn(i))
				cache.Get(key)
			}
		}
	})

	elapsed := time.Since(start)
	qps := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(qps, "ops/sec")
}
