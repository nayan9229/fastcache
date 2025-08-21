package fastcache

import (
	"fmt"
	"sync/atomic"
)

// Stats represents cache statistics
type Stats struct {
	TotalSize     int64   `json:"total_size"`
	TotalEntries  int64   `json:"total_entries"`
	HitCount      int64   `json:"hit_count"`
	MissCount     int64   `json:"miss_count"`
	HitRatio      float64 `json:"hit_ratio"`
	MemoryUsage   string  `json:"memory_usage"`
	ShardCount    int     `json:"shard_count"`
	MaxMemory     int64   `json:"max_memory"`
	MemoryPercent float64 `json:"memory_percent"`
}

// GetStats returns current cache statistics
func (c *Cache) GetStats() *Stats {
	totalEntries := int64(0)
	for _, shard := range c.shards {
		shard.mu.RLock()
		totalEntries += int64(len(shard.data))
		shard.mu.RUnlock()
	}

	hits := atomic.LoadInt64(&c.totalHits)
	misses := atomic.LoadInt64(&c.totalMiss)
	total := hits + misses

	var hitRatio float64
	if total > 0 {
		hitRatio = float64(hits) / float64(total)
	}

	size := atomic.LoadInt64(&c.totalSize)
	memoryPercent := float64(size) / float64(c.config.MaxMemoryBytes) * 100

	return &Stats{
		TotalSize:     size,
		TotalEntries:  totalEntries,
		HitCount:      hits,
		MissCount:     misses,
		HitRatio:      hitRatio,
		MemoryUsage:   formatBytes(size),
		ShardCount:    c.config.ShardCount,
		MaxMemory:     c.config.MaxMemoryBytes,
		MemoryPercent: memoryPercent,
	}
}

// ShardStats represents statistics for a single shard
type ShardStats struct {
	ShardID     int     `json:"shard_id"`
	EntryCount  int     `json:"entry_count"`
	Size        int64   `json:"size"`
	HitCount    int64   `json:"hit_count"`
	MissCount   int64   `json:"miss_count"`
	HitRatio    float64 `json:"hit_ratio"`
	MemoryUsage string  `json:"memory_usage"`
}

// GetShardStats returns statistics for all shards
func (c *Cache) GetShardStats() []ShardStats {
	stats := make([]ShardStats, len(c.shards))

	for i, shard := range c.shards {
		shard.mu.RLock()
		entryCount := len(shard.data)
		size := atomic.LoadInt64(&shard.size)
		hits := atomic.LoadInt64(&shard.hitCount)
		misses := atomic.LoadInt64(&shard.missCount)
		shard.mu.RUnlock()

		total := hits + misses
		var hitRatio float64
		if total > 0 {
			hitRatio = float64(hits) / float64(total)
		}

		stats[i] = ShardStats{
			ShardID:     i,
			EntryCount:  entryCount,
			Size:        size,
			HitCount:    hits,
			MissCount:   misses,
			HitRatio:    hitRatio,
			MemoryUsage: formatBytes(size),
		}
	}

	return stats
}

// ResetStats resets all statistics counters
func (c *Cache) ResetStats() {
	atomic.StoreInt64(&c.totalHits, 0)
	atomic.StoreInt64(&c.totalMiss, 0)

	for _, shard := range c.shards {
		atomic.StoreInt64(&shard.hitCount, 0)
		atomic.StoreInt64(&shard.missCount, 0)
	}
}

// MemoryInfo provides detailed memory information
type MemoryInfo struct {
	Used               int64   `json:"used"`
	UsedFormatted      string  `json:"used_formatted"`
	Max                int64   `json:"max"`
	MaxFormatted       string  `json:"max_formatted"`
	Available          int64   `json:"available"`
	AvailableFormatted string  `json:"available_formatted"`
	Percent            float64 `json:"percent"`
	ShardSizes         []int64 `json:"shard_sizes"`
}

// GetMemoryInfo returns detailed memory usage information
func (c *Cache) GetMemoryInfo() *MemoryInfo {
	used := atomic.LoadInt64(&c.totalSize)
	available := c.config.MaxMemoryBytes - used
	if available < 0 {
		available = 0
	}
	percent := float64(used) / float64(c.config.MaxMemoryBytes) * 100

	shardSizes := make([]int64, len(c.shards))
	for i, shard := range c.shards {
		shardSizes[i] = atomic.LoadInt64(&shard.size)
	}

	return &MemoryInfo{
		Used:               used,
		UsedFormatted:      formatBytes(used),
		Max:                c.config.MaxMemoryBytes,
		MaxFormatted:       formatBytes(c.config.MaxMemoryBytes),
		Available:          available,
		AvailableFormatted: formatBytes(available),
		Percent:            percent,
		ShardSizes:         shardSizes,
	}
}

// PerformanceMetrics provides performance-related metrics
type PerformanceMetrics struct {
	TotalOperations int64   `json:"total_operations"`
	HitRate         float64 `json:"hit_rate"`
	MissRate        float64 `json:"miss_rate"`
	AvgShardLoad    float64 `json:"avg_shard_load"`
	MaxShardLoad    int     `json:"max_shard_load"`
	MinShardLoad    int     `json:"min_shard_load"`
	LoadBalance     float64 `json:"load_balance"` // Standard deviation of shard loads
}

// GetPerformanceMetrics returns performance metrics
func (c *Cache) GetPerformanceMetrics() *PerformanceMetrics {
	hits := atomic.LoadInt64(&c.totalHits)
	misses := atomic.LoadInt64(&c.totalMiss)
	total := hits + misses

	var hitRate, missRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total)
		missRate = float64(misses) / float64(total)
	}

	// Calculate shard load distribution
	var totalEntries int64
	var maxLoad, minLoad int
	loads := make([]int, len(c.shards))

	for i, shard := range c.shards {
		shard.mu.RLock()
		load := len(shard.data)
		loads[i] = load
		totalEntries += int64(load)

		if i == 0 || load > maxLoad {
			maxLoad = load
		}
		if i == 0 || load < minLoad {
			minLoad = load
		}
		shard.mu.RUnlock()
	}

	avgLoad := float64(totalEntries) / float64(len(c.shards))

	// Calculate standard deviation for load balance
	var variance float64
	for _, load := range loads {
		diff := float64(load) - avgLoad
		variance += diff * diff
	}
	variance /= float64(len(loads))
	loadBalance := variance // Using variance as load balance metric

	return &PerformanceMetrics{
		TotalOperations: total,
		HitRate:         hitRate,
		MissRate:        missRate,
		AvgShardLoad:    avgLoad,
		MaxShardLoad:    maxLoad,
		MinShardLoad:    minLoad,
		LoadBalance:     loadBalance,
	}
}

// formatBytes formats bytes into human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// String returns a human-readable representation of the stats
func (s *Stats) String() string {
	return fmt.Sprintf("Entries: %d, Memory: %s (%.1f%%), Hit Ratio: %.2f%%, Operations: %d",
		s.TotalEntries, s.MemoryUsage, s.MemoryPercent, s.HitRatio*100, s.HitCount+s.MissCount)
}
