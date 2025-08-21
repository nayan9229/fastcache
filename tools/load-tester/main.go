package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/nayan9229/fastcache"
)

// LoadTestConfig defines the load test parameters
type LoadTestConfig struct {
	Duration       time.Duration `json:"duration"`
	Workers        int           `json:"workers"`
	TargetQPS      int           `json:"target_qps"`
	WriteRatio     float64       `json:"write_ratio"`
	DeleteRatio    float64       `json:"delete_ratio"`
	KeyRange       int           `json:"key_range"`
	ValueSize      int           `json:"value_size"`
	WarmupDuration time.Duration `json:"warmup_duration"`
	CooldownPeriod time.Duration `json:"cooldown_period"`
	MemoryLimit    int64         `json:"memory_limit_mb"`
	ShardCount     int           `json:"shard_count"`
	ReportInterval time.Duration `json:"report_interval"`
}

// LoadTestResults contains the test results
type LoadTestResults struct {
	Config          LoadTestConfig `json:"config"`
	StartTime       time.Time      `json:"start_time"`
	EndTime         time.Time      `json:"end_time"`
	Duration        time.Duration  `json:"duration"`
	TotalOperations int64          `json:"total_operations"`
	ActualQPS       float64        `json:"actual_qps"`
	Sets            int64          `json:"sets"`
	Gets            int64          `json:"gets"`
	Deletes         int64          `json:"deletes"`
	Hits            int64          `json:"hits"`
	Misses          int64          `json:"misses"`
	HitRatio        float64        `json:"hit_ratio"`
	Errors          int64          `json:"errors"`

	// Latency statistics (in nanoseconds)
	LatencyStats LatencyStats `json:"latency_stats"`

	// Cache statistics at end
	FinalCacheStats *fastcache.Stats `json:"final_cache_stats"`

	// System metrics
	SystemMetrics SystemMetrics `json:"system_metrics"`
}

// LatencyStats contains latency measurements
type LatencyStats struct {
	Min     time.Duration `json:"min"`
	Max     time.Duration `json:"max"`
	Mean    time.Duration `json:"mean"`
	P50     time.Duration `json:"p50"`
	P95     time.Duration `json:"p95"`
	P99     time.Duration `json:"p99"`
	Samples []int64       `json:"-"` // Don't serialize raw samples
}

// SystemMetrics contains system performance data
type SystemMetrics struct {
	StartMemory   runtime.MemStats `json:"start_memory"`
	EndMemory     runtime.MemStats `json:"end_memory"`
	PeakMemory    uint64           `json:"peak_memory"`
	GCRuns        uint32           `json:"gc_runs"`
	MaxGoroutines int              `json:"max_goroutines"`
}

// WorkerStats tracks per-worker statistics
type WorkerStats struct {
	Sets      int64
	Gets      int64
	Deletes   int64
	Hits      int64
	Misses    int64
	Errors    int64
	Latencies []int64 // Nanoseconds
}

var (
	// Command line flags
	duration       = flag.Duration("duration", 30*time.Second, "Test duration")
	workers        = flag.Int("workers", 100, "Number of worker goroutines")
	targetQPS      = flag.Int("qps", 10000, "Target operations per second")
	writeRatio     = flag.Float64("write-ratio", 0.3, "Ratio of write operations (0.0-1.0)")
	deleteRatio    = flag.Float64("delete-ratio", 0.05, "Ratio of delete operations (0.0-1.0)")
	keyRange       = flag.Int("key-range", 10000, "Range of keys to operate on")
	valueSize      = flag.Int("value-size", 100, "Size of values in bytes")
	warmupDuration = flag.Duration("warmup", 5*time.Second, "Warmup duration")
	memoryLimitMB  = flag.Int64("memory-limit", 512, "Memory limit in MB")
	shardCount     = flag.Int("shards", 1024, "Number of cache shards")
	reportInterval = flag.Duration("report-interval", 5*time.Second, "Progress report interval")
	outputFile     = flag.String("output", "", "Output file for results (JSON)")
	profile        = flag.Bool("profile", false, "Enable profiling")
	verbose        = flag.Bool("verbose", false, "Verbose output")
)

func main() {
	flag.Parse()

	fmt.Println("🚀 FastCache Load Tester")
	fmt.Println("========================")

	config := LoadTestConfig{
		Duration:       *duration,
		Workers:        *workers,
		TargetQPS:      *targetQPS,
		WriteRatio:     *writeRatio,
		DeleteRatio:    *deleteRatio,
		KeyRange:       *keyRange,
		ValueSize:      *valueSize,
		WarmupDuration: *warmupDuration,
		MemoryLimit:    *memoryLimitMB,
		ShardCount:     *shardCount,
		ReportInterval: *reportInterval,
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Print test configuration
	printConfig(config)

	// Run the load test
	results, err := runLoadTest(config)
	if err != nil {
		log.Fatalf("Load test failed: %v", err)
	}

	// Print results
	printResults(results)

	// Save results to file if specified
	if *outputFile != "" {
		saveResults(results, *outputFile)
	}

	// Performance assessment
	assessPerformance(results)
}

func validateConfig(config LoadTestConfig) error {
	if config.WriteRatio+config.DeleteRatio > 1.0 {
		return fmt.Errorf("write ratio + delete ratio cannot exceed 1.0")
	}
	if config.Workers <= 0 || config.TargetQPS <= 0 {
		return fmt.Errorf("workers and target QPS must be positive")
	}
	if config.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	return nil
}

func printConfig(config LoadTestConfig) {
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Duration: %v\n", config.Duration)
	fmt.Printf("  Workers: %d\n", config.Workers)
	fmt.Printf("  Target QPS: %d\n", config.TargetQPS)
	fmt.Printf("  Write Ratio: %.1f%%\n", config.WriteRatio*100)
	fmt.Printf("  Delete Ratio: %.1f%%\n", config.DeleteRatio*100)
	fmt.Printf("  Read Ratio: %.1f%%\n", (1-config.WriteRatio-config.DeleteRatio)*100)
	fmt.Printf("  Key Range: %d\n", config.KeyRange)
	fmt.Printf("  Value Size: %d bytes\n", config.ValueSize)
	fmt.Printf("  Memory Limit: %d MB\n", config.MemoryLimit)
	fmt.Printf("  Shard Count: %d\n", config.ShardCount)
	fmt.Printf("  Warmup: %v\n", config.WarmupDuration)
	fmt.Println()
}

func runLoadTest(config LoadTestConfig) (*LoadTestResults, error) {
	// Create cache
	cacheConfig := &fastcache.Config{
		MaxMemoryBytes:  config.MemoryLimit * 1024 * 1024,
		ShardCount:      config.ShardCount,
		DefaultTTL:      10 * time.Minute,
		CleanupInterval: time.Minute,
	}

	cache := fastcache.New(cacheConfig)
	defer cache.Close()

	// Initialize results
	results := &LoadTestResults{
		Config:    config,
		StartTime: time.Now(),
	}

	// Capture initial system metrics
	runtime.ReadMemStats(&results.SystemMetrics.StartMemory)

	// Warmup phase
	if config.WarmupDuration > 0 {
		fmt.Printf("Warming up for %v...\n", config.WarmupDuration)
		runWarmup(cache, config)
	}

	// Prepare workers
	workerStats := make([]WorkerStats, config.Workers)
	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// Calculate operations per worker to achieve target QPS
	opsPerWorkerPerSecond := config.TargetQPS / config.Workers
	if opsPerWorkerPerSecond < 1 {
		opsPerWorkerPerSecond = 1
	}

	fmt.Printf("Starting load test with %d workers...\n", config.Workers)
	fmt.Printf("Target: %d ops/worker/sec = %d total QPS\n", opsPerWorkerPerSecond, opsPerWorkerPerSecond*config.Workers)

	// Start progress reporting
	go reportProgress(cache, config.ReportInterval, stopCh, results)

	// Start system monitoring
	go monitorSystem(&results.SystemMetrics, stopCh)

	// Start workers
	actualStartTime := time.Now()
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go worker(cache, config, &workerStats[i], opsPerWorkerPerSecond, &wg, stopCh)
	}

	// Wait for test duration
	time.Sleep(config.Duration)
	close(stopCh)
	wg.Wait()

	results.EndTime = time.Now()
	results.Duration = results.EndTime.Sub(actualStartTime)

	// Aggregate results
	aggregateResults(results, workerStats, cache)

	// Capture final system metrics
	runtime.ReadMemStats(&results.SystemMetrics.EndMemory)
	results.SystemMetrics.GCRuns = results.SystemMetrics.EndMemory.NumGC - results.SystemMetrics.StartMemory.NumGC

	return results, nil
}

func runWarmup(cache *fastcache.Cache, config LoadTestConfig) {
	// Pre-populate cache with some data
	for i := 0; i < config.KeyRange/2; i++ {
		key := fmt.Sprintf("warmup_key_%d", i)
		value := generateValue(config.ValueSize)
		cache.Set(key, value)
	}

	// Run brief workload
	numWorkers := config.Workers / 4 // Use fewer workers for warmup
	if numWorkers < 1 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					key := fmt.Sprintf("warmup_key_%d", rand.Intn(config.KeyRange))
					if rand.Float64() < 0.7 {
						cache.Get(key)
					} else {
						value := generateValue(config.ValueSize)
						cache.Set(key, value)
					}
				}
			}
		}()
	}

	time.Sleep(config.WarmupDuration)
	close(stopCh)
	wg.Wait()
}

func worker(cache *fastcache.Cache, config LoadTestConfig, stats *WorkerStats, targetOpsPerSec int, wg *sync.WaitGroup, stopCh <-chan struct{}) {
	defer wg.Done()

	rand.Seed(time.Now().UnixNano() + int64(uintptr(unsafe.Pointer(stats))))

	// Calculate timing for target QPS
	targetInterval := time.Second / time.Duration(targetOpsPerSec)
	ticker := time.NewTicker(targetInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			performOperation(cache, config, stats)
		}
	}
}

func performOperation(cache *fastcache.Cache, config LoadTestConfig, stats *WorkerStats) {
	start := time.Now()

	r := rand.Float64()

	if r < config.DeleteRatio {
		// Delete operation
		key := fmt.Sprintf("key_%d", rand.Intn(config.KeyRange))
		cache.Delete(key)
		atomic.AddInt64(&stats.Deletes, 1)

	} else if r < config.WriteRatio+config.DeleteRatio {
		// Write operation
		key := fmt.Sprintf("key_%d", rand.Intn(config.KeyRange))
		value := generateValue(config.ValueSize)
		err := cache.Set(key, value)
		if err != nil {
			atomic.AddInt64(&stats.Errors, 1)
		} else {
			atomic.AddInt64(&stats.Sets, 1)
		}

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

	// Record latency
	latency := time.Since(start).Nanoseconds()
	stats.Latencies = append(stats.Latencies, latency)
}

func generateValue(size int) []byte {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, size)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return result
}

func reportProgress(cache *fastcache.Cache, interval time.Duration, stopCh <-chan struct{}, results *LoadTestResults) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lastTime := time.Now()
	var lastOps int64

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			stats := cache.GetStats()
			currentOps := stats.HitCount + stats.MissCount
			currentTime := time.Now()

			// Calculate QPS since last report
			deltaOps := currentOps - lastOps
			deltaTime := currentTime.Sub(lastTime).Seconds()
			currentQPS := float64(deltaOps) / deltaTime

			fmt.Printf("Progress: %d ops total, %.0f QPS, %.2f%% hit ratio, %s memory\n",
				currentOps, currentQPS, stats.HitRatio*100, stats.MemoryUsage)

			lastOps = currentOps
			lastTime = currentTime
		}
	}
}

func monitorSystem(sysMetrics *SystemMetrics, stopCh <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			goroutines := runtime.NumGoroutine()
			if goroutines > sysMetrics.MaxGoroutines {
				sysMetrics.MaxGoroutines = goroutines
			}

			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			if m.HeapAlloc > sysMetrics.PeakMemory {
				sysMetrics.PeakMemory = m.HeapAlloc
			}
		}
	}
}

func aggregateResults(results *LoadTestResults, workerStats []WorkerStats, cache *fastcache.Cache) {
	var allLatencies []int64

	for _, stats := range workerStats {
		results.Sets += stats.Sets
		results.Gets += stats.Gets
		results.Deletes += stats.Deletes
		results.Hits += stats.Hits
		results.Misses += stats.Misses
		results.Errors += stats.Errors
		allLatencies = append(allLatencies, stats.Latencies...)
	}

	results.TotalOperations = results.Sets + results.Gets + results.Deletes
	results.ActualQPS = float64(results.TotalOperations) / results.Duration.Seconds()

	if results.Hits+results.Misses > 0 {
		results.HitRatio = float64(results.Hits) / float64(results.Hits+results.Misses)
	}

	// Calculate latency statistics
	if len(allLatencies) > 0 {
		results.LatencyStats = calculateLatencyStats(allLatencies)
	}

	// Get final cache statistics
	results.FinalCacheStats = cache.GetStats()
}

func calculateLatencyStats(latencies []int64) LatencyStats {
	if len(latencies) == 0 {
		return LatencyStats{}
	}

	// Sort latencies for percentile calculation
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	var sum int64
	for _, lat := range latencies {
		sum += lat
	}

	stats := LatencyStats{
		Min:     time.Duration(latencies[0]),
		Max:     time.Duration(latencies[len(latencies)-1]),
		Mean:    time.Duration(sum / int64(len(latencies))),
		P50:     time.Duration(latencies[len(latencies)*50/100]),
		P95:     time.Duration(latencies[len(latencies)*95/100]),
		P99:     time.Duration(latencies[len(latencies)*99/100]),
		Samples: latencies,
	}

	return stats
}

func printResults(results *LoadTestResults) {
	fmt.Println("\n🎯 Load Test Results")
	fmt.Println("===================")
	fmt.Printf("Duration: %v\n", results.Duration)
	fmt.Printf("Total Operations: %d\n", results.TotalOperations)
	fmt.Printf("Actual QPS: %.0f\n", results.ActualQPS)
	fmt.Printf("Target QPS: %d (%.1f%% achieved)\n",
		results.Config.TargetQPS,
		results.ActualQPS/float64(results.Config.TargetQPS)*100)

	fmt.Println("\nOperation Breakdown:")
	fmt.Printf("  SET: %d (%.0f/sec)\n", results.Sets, float64(results.Sets)/results.Duration.Seconds())
	fmt.Printf("  GET: %d (%.0f/sec)\n", results.Gets, float64(results.Gets)/results.Duration.Seconds())
	fmt.Printf("  DELETE: %d (%.0f/sec)\n", results.Deletes, float64(results.Deletes)/results.Duration.Seconds())
	fmt.Printf("  Errors: %d\n", results.Errors)

	fmt.Println("\nCache Performance:")
	fmt.Printf("  Hits: %d\n", results.Hits)
	fmt.Printf("  Misses: %d\n", results.Misses)
	fmt.Printf("  Hit Ratio: %.2f%%\n", results.HitRatio*100)

	if results.FinalCacheStats != nil {
		fmt.Printf("  Final Entries: %d\n", results.FinalCacheStats.TotalEntries)
		fmt.Printf("  Memory Usage: %s\n", results.FinalCacheStats.MemoryUsage)
	}

	fmt.Println("\nLatency Statistics:")
	fmt.Printf("  Min: %v\n", results.LatencyStats.Min)
	fmt.Printf("  Mean: %v\n", results.LatencyStats.Mean)
	fmt.Printf("  P50: %v\n", results.LatencyStats.P50)
	fmt.Printf("  P95: %v\n", results.LatencyStats.P95)
	fmt.Printf("  P99: %v\n", results.LatencyStats.P99)
	fmt.Printf("  Max: %v\n", results.LatencyStats.Max)

	fmt.Println("\nSystem Metrics:")
	fmt.Printf("  Peak Memory: %.2f MB\n", float64(results.SystemMetrics.PeakMemory)/1024/1024)
	fmt.Printf("  GC Runs: %d\n", results.SystemMetrics.GCRuns)
	fmt.Printf("  Max Goroutines: %d\n", results.SystemMetrics.MaxGoroutines)
}

func saveResults(results *LoadTestResults, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Printf("Failed to create output file: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		log.Printf("Failed to write results: %v", err)
	} else {
		fmt.Printf("\nResults saved to: %s\n", filename)
	}
}

func assessPerformance(results *LoadTestResults) {
	fmt.Println("\n📊 Performance Assessment")
	fmt.Println("========================")

	// QPS assessment
	qpsAchievement := results.ActualQPS / float64(results.Config.TargetQPS)
	if qpsAchievement >= 0.95 {
		fmt.Printf("✅ QPS Target: EXCELLENT (%.1f%% achieved)\n", qpsAchievement*100)
	} else if qpsAchievement >= 0.8 {
		fmt.Printf("✅ QPS Target: GOOD (%.1f%% achieved)\n", qpsAchievement*100)
	} else if qpsAchievement >= 0.6 {
		fmt.Printf("⚠️  QPS Target: MODERATE (%.1f%% achieved)\n", qpsAchievement*100)
	} else {
		fmt.Printf("❌ QPS Target: POOR (%.1f%% achieved)\n", qpsAchievement*100)
	}

	// Hit ratio assessment
	if results.HitRatio >= 0.9 {
		fmt.Printf("✅ Hit Ratio: EXCELLENT (%.1f%%)\n", results.HitRatio*100)
	} else if results.HitRatio >= 0.7 {
		fmt.Printf("✅ Hit Ratio: GOOD (%.1f%%)\n", results.HitRatio*100)
	} else if results.HitRatio >= 0.5 {
		fmt.Printf("⚠️  Hit Ratio: MODERATE (%.1f%%)\n", results.HitRatio*100)
	} else {
		fmt.Printf("❌ Hit Ratio: POOR (%.1f%%)\n", results.HitRatio*100)
	}

	// Latency assessment
	if results.LatencyStats.P95 < time.Microsecond {
		fmt.Printf("✅ Latency: EXCELLENT (P95: %v)\n", results.LatencyStats.P95)
	} else if results.LatencyStats.P95 < 10*time.Microsecond {
		fmt.Printf("✅ Latency: GOOD (P95: %v)\n", results.LatencyStats.P95)
	} else if results.LatencyStats.P95 < 100*time.Microsecond {
		fmt.Printf("⚠️  Latency: MODERATE (P95: %v)\n", results.LatencyStats.P95)
	} else {
		fmt.Printf("❌ Latency: POOR (P95: %v)\n", results.LatencyStats.P95)
	}

	// Error rate assessment
	errorRate := float64(results.Errors) / float64(results.TotalOperations)
	if errorRate == 0 {
		fmt.Printf("✅ Error Rate: PERFECT (0%%)\n")
	} else if errorRate < 0.001 {
		fmt.Printf("✅ Error Rate: EXCELLENT (%.3f%%)\n", errorRate*100)
	} else if errorRate < 0.01 {
		fmt.Printf("⚠️  Error Rate: ACCEPTABLE (%.3f%%)\n", errorRate*100)
	} else {
		fmt.Printf("❌ Error Rate: HIGH (%.3f%%)\n", errorRate*100)
	}

	fmt.Println("\nRecommendations:")
	if qpsAchievement < 0.8 {
		fmt.Println("- Consider increasing shard count or optimizing workload")
	}
	if results.HitRatio < 0.7 {
		fmt.Println("- Consider increasing cache size or adjusting TTL")
	}
	if results.LatencyStats.P95 > 10*time.Microsecond {
		fmt.Println("- Consider reducing value size or optimizing operations")
	}
	if errorRate > 0.001 {
		fmt.Println("- Investigate error causes and optimize error handling")
	}
}
