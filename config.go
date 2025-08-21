package fastcache

import "time"

// Config holds configuration for the cache
type Config struct {
	// MaxMemoryBytes is the maximum memory usage before eviction starts (e.g., 512MB)
	MaxMemoryBytes int64

	// ShardCount is the number of shards for concurrent access
	// Higher values reduce lock contention but increase memory overhead
	ShardCount int

	// DefaultTTL is the default time-to-live for entries
	// Set to 0 for no expiration
	DefaultTTL time.Duration

	// CleanupInterval determines how often expired entries are cleaned up
	CleanupInterval time.Duration
}

// DefaultConfig returns a default configuration optimized for 1M QPS
func DefaultConfig() *Config {
	return &Config{
		MaxMemoryBytes:  512 * 1024 * 1024, // 512MB
		ShardCount:      1024,              // High shard count for concurrency
		DefaultTTL:      time.Hour,         // 1 hour default TTL
		CleanupInterval: time.Minute,       // Cleanup every minute
	}
}

// HighConcurrencyConfig returns a configuration optimized for very high concurrency
func HighConcurrencyConfig() *Config {
	return &Config{
		MaxMemoryBytes:  1024 * 1024 * 1024, // 1GB
		ShardCount:      2048,               // Very high shard count
		DefaultTTL:      30 * time.Minute,   // 30 minutes default TTL
		CleanupInterval: 30 * time.Second,   // More frequent cleanup
	}
}

// LowMemoryConfig returns a configuration for memory-constrained environments
func LowMemoryConfig() *Config {
	return &Config{
		MaxMemoryBytes:  64 * 1024 * 1024, // 64MB
		ShardCount:      256,              // Lower shard count
		DefaultTTL:      15 * time.Minute, // Shorter TTL to free memory faster
		CleanupInterval: 30 * time.Second, // More frequent cleanup
	}
}

// CustomConfig creates a configuration with custom parameters
func CustomConfig(maxMemoryMB int, shardCount int, defaultTTL time.Duration) *Config {
	return &Config{
		MaxMemoryBytes:  int64(maxMemoryMB) * 1024 * 1024,
		ShardCount:      shardCount,
		DefaultTTL:      defaultTTL,
		CleanupInterval: time.Minute,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.MaxMemoryBytes <= 0 {
		return ErrInvalidConfig{Field: "MaxMemoryBytes", Message: "must be greater than 0"}
	}

	if c.ShardCount <= 0 {
		return ErrInvalidConfig{Field: "ShardCount", Message: "must be greater than 0"}
	}

	if c.ShardCount > 65536 {
		return ErrInvalidConfig{Field: "ShardCount", Message: "must be less than 65536"}
	}

	if c.CleanupInterval <= 0 {
		return ErrInvalidConfig{Field: "CleanupInterval", Message: "must be greater than 0"}
	}

	return nil
}
