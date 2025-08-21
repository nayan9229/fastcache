package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nayan9229/fastcache"
)

// OperationStats tracks performance metrics
type OperationStats struct {
	Sets    int64
	Gets    int64
	Deletes int64
	Hits    int64
	Misses  int64
}

// WorkerConfig defines worker behavior
type WorkerConfig struct {
	ID          int
	Operations  int
	WriteRatio  float32       // Percentage of write operations
	DeleteRatio float32       // Percentage of delete operations
	KeyRange    int           // Range of keys to operate on
	ValueSize   int           // Size of values in bytes
	ThinkTime   time.Duration // Delay between operations
}

func main() {
	fmt.Println("=== FastCache High Concurrency Test ===")

	// Show system information
	showSystemInfo()

	// Run different concurrency scenarios
	scenarios := []struct {
		name        string
		workers     int
		operations  int
		writeRatio  float32
		deleteRatio float32
		keyRange    int
	}{
		{"Light Load", 50, 1000, 0.3, 0.1, 1000},
		{"Medium Load", 200, 2000, 0.3, 0.1, 2000},
		{"Heavy Load", 500, 3000, 0.2, 0.05, 5000},
		{"Extreme Load", 1000, 5000, 0.1, 0.02, 10000},
	}

	for _, scenario := range scenarios {
		fmt.Printf("Running %s scenario...\n", scenario.name)
		runConcurrencyTest(scenario.workers, scenario.operations,
			scenario.writeRatio, scenario.deleteRatio, scenario.keyRange)
		fmt.Println()
	}

	// Run sustained load test
	fmt.Println("Running sustained load test...")
	runSustainedLoadTest()
}

func showSystemInfo() {
	fmt.Printf("System Information:\n")
	fmt.Printf("- Go Version: %s\n", runtime.Version())
	fmt.Printf("- CPU Cores: %d\n", runtime.NumCPU())
	fmt.Printf("- GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("- Total Memory: %.2f MB\n", float64(m.Sys)/1024/1024)
	fmt.Println()
}

func runConcurrencyTest(numWorkers, operations int, writeRatio, deleteRatio float32, keyRange int) {
	// Create cache with high-concurrency configuration
	config := &fastcache.Config{
		MaxMemoryBytes:  256 * 1024 * 1024, // 256MB
		ShardCount:      1024,              // High shard count
		DefaultTTL:      10 * time.Minute,
		CleanupInterval: time.Minute,
	}

	cache := fastcache.New(config)
	defer cache.Close()

	// Pre-populate cache
	fmt.Printf("Pre-populating cache with %d entries...\n", keyRange/2)
	for i := 0; i < keyRange/2; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := generateRandomValue(100)
		cache.Set(key, value)
	}

	var stats OperationStats
	var wg sync.WaitGroup

	start := time.Now()

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(cache, WorkerConfig{
			ID:          i,
			Operations:  operations,
			WriteRatio:  writeRatio,
			DeleteRatio: deleteRatio,
			KeyRange:    keyRange,
			ValueSize:   100,
			ThinkTime:   0, // No delay for stress test
		}, &stats, &wg)
	}

	// Monitor progress
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Progress monitoring
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	lastSets := int64(0)
	lastGets := int64(0)

	for {
		select {
		case <-done:
			ticker.Stop()
			goto finished
		case <-ticker.C:
			currentSets := atomic.LoadInt64(&stats.Sets)
			currentGets := atomic.LoadInt64(&stats.Gets)

			setsPerSec := currentSets - lastSets
			getsPerSec := currentGets - lastGets

			fmt.Printf("Progress: SET %d/sec, GET %d/sec\n", setsPerSec, getsPerSec)

			lastSets = currentSets
			lastGets = currentGets
		}
	}

finished:
	duration := time.Since(start)

	// Get final cache statistics
	cacheStats := cache.GetStats()

	// Calculate results
	totalOps := atomic.LoadInt64(&stats.Sets) + atomic.LoadInt64(&stats.Gets) + atomic.LoadInt64(&stats.Deletes)
	opsPerSecond := float64(totalOps) / duration.Seconds()

	fmt.Printf("Results for %d workers, %d ops each:\n", numWorkers, operations)
	fmt.Printf("- Duration: %v\n", duration)
	fmt.Printf("- Total Operations: %d\n", totalOps)
	fmt.Printf("- Operations/sec: %.0f\n", opsPerSecond)
	fmt.Printf("- SET Operations: %d\n", atomic.LoadInt64(&stats.Sets))
	fmt.Printf("- GET Operations: %d\n", atomic.LoadInt64(&stats.Gets))
	fmt.Printf("- DELETE Operations: %d\n", atomic.LoadInt64(&stats.Deletes))
	fmt.Printf("- Cache Hits: %d\n", cacheStats.HitCount)
	fmt.Printf("- Cache Misses: %d\n", cacheStats.MissCount)
	fmt.Printf("- Hit Ratio: %.2f%%\n", cacheStats.HitRatio*100)
	fmt.Printf("- Memory Usage: %s\n", cacheStats.MemoryUsage)
	fmt.Printf("- Cache Entries: %d\n", cacheStats.TotalEntries)
}

func worker(cache *fastcache.Cache, config WorkerConfig, stats *OperationStats, wg *sync.WaitGroup) {
	defer wg.Done()

	rand.New(rand.NewSource(int64(config.ID)))

	for i := 0; i < config.Operations; i++ {
		if config.ThinkTime > 0 {
			time.Sleep(config.ThinkTime)
		}

		// Determine operation type
		r := rand.Float32()

		if r < config.DeleteRatio {
			// Delete operation
			key := fmt.Sprintf("key_%d", rand.Intn(config.KeyRange))
			cache.Delete(key)
			atomic.AddInt64(&stats.Deletes, 1)

		} else if r < config.WriteRatio {
			// Write operation
			key := fmt.Sprintf("key_%d", rand.Intn(config.KeyRange))
			value := generateRandomValue(config.ValueSize)
			cache.Set(key, value)
			atomic.AddInt64(&stats.Sets, 1)

		} else {
			// Read operation
			key := fmt.Sprintf("key_%d", rand.Intn(config.KeyRange))
			if _, exists := cache.Get(key); exists {
				atomic.AddInt64(&stats.Hits, 1)
			} else {
				atomic.AddInt64(&stats.Misses, 1)
			}
			atomic.AddInt64(&stats.Gets, 1)
		}
	}
}

func runSustainedLoadTest() {
	config := fastcache.HighConcurrencyConfig()
	cache := fastcache.New(config)
	defer cache.Close()

	const duration = 30 * time.Second
	const numWorkers = 500
	const targetQPS = 100000 // Target 100K QPS

	// Calculate operations per worker
	// opsPerWorker := int(targetQPS * int(duration.Seconds()) / numWorkers)

	fmt.Printf("Sustained load test: %d workers, %.0fs duration, target %d QPS\n",
		numWorkers, duration.Seconds(), targetQPS)

	var stats OperationStats
	var wg sync.WaitGroup

	start := time.Now()
	stop := make(chan struct{})

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go sustainedWorker(cache, i, &stats, &wg, stop)
	}

	// Stop workers after duration
	time.Sleep(duration)
	close(stop)
	wg.Wait()

	elapsed := time.Since(start)

	// Calculate final results
	totalOps := atomic.LoadInt64(&stats.Sets) + atomic.LoadInt64(&stats.Gets) + atomic.LoadInt64(&stats.Deletes)
	actualQPS := float64(totalOps) / elapsed.Seconds()

	cacheStats := cache.GetStats()

	fmt.Printf("\nSustained Load Test Results:\n")
	fmt.Printf("- Duration: %v\n", elapsed)
	fmt.Printf("- Total Operations: %d\n", totalOps)
	fmt.Printf("- Actual QPS: %.0f\n", actualQPS)
	fmt.Printf("- Target QPS: %d\n", targetQPS)
	fmt.Printf("- QPS Achievement: %.1f%%\n", (actualQPS/float64(targetQPS))*100)
	fmt.Printf("- Hit Ratio: %.2f%%\n", cacheStats.HitRatio*100)
	fmt.Printf("- Memory Usage: %s\n", cacheStats.MemoryUsage)
	fmt.Printf("- Cache Entries: %d\n", cacheStats.TotalEntries)

	// Performance analysis
	if actualQPS >= float64(targetQPS)*0.9 {
		fmt.Printf("✅ Performance target achieved!\n")
	} else if actualQPS >= float64(targetQPS)*0.7 {
		fmt.Printf("⚠️  Performance target partially achieved\n")
	} else {
		fmt.Printf("❌ Performance target not achieved\n")
	}

	// Show performance per operation type
	fmt.Printf("\nOperation Breakdown:\n")
	fmt.Printf("- SET: %d (%.0f/sec)\n",
		atomic.LoadInt64(&stats.Sets),
		float64(atomic.LoadInt64(&stats.Sets))/elapsed.Seconds())
	fmt.Printf("- GET: %d (%.0f/sec)\n",
		atomic.LoadInt64(&stats.Gets),
		float64(atomic.LoadInt64(&stats.Gets))/elapsed.Seconds())
	fmt.Printf("- DELETE: %d (%.0f/sec)\n",
		atomic.LoadInt64(&stats.Deletes),
		float64(atomic.LoadInt64(&stats.Deletes))/elapsed.Seconds())
}

func sustainedWorker(cache *fastcache.Cache, workerID int, stats *OperationStats, wg *sync.WaitGroup, stop <-chan struct{}) {
	defer wg.Done()

	rand.New(rand.NewSource(int64(workerID)))

	for {
		select {
		case <-stop:
			return
		default:
			// Mix of operations: 20% writes, 75% reads, 5% deletes
			r := rand.Float32()

			if r < 0.05 {
				// Delete
				key := fmt.Sprintf("sustained_key_%d", rand.Intn(10000))
				cache.Delete(key)
				atomic.AddInt64(&stats.Deletes, 1)

			} else if r < 0.25 {
				// Write
				key := fmt.Sprintf("sustained_key_%d", rand.Intn(10000))
				value := generateRandomValue(50)
				cache.Set(key, value)
				atomic.AddInt64(&stats.Sets, 1)

			} else {
				// Read
				key := fmt.Sprintf("sustained_key_%d", rand.Intn(10000))
				if _, exists := cache.Get(key); exists {
					atomic.AddInt64(&stats.Hits, 1)
				} else {
					atomic.AddInt64(&stats.Misses, 1)
				}
				atomic.AddInt64(&stats.Gets, 1)
			}
		}
	}
}

func generateRandomValue(size int) []byte {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, size)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return result
}
