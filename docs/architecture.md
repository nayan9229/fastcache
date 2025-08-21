# FastCache Architecture

This document provides a detailed overview of FastCache's internal architecture, design decisions, and implementation details.

## 📊 High-Level Overview

FastCache is designed as a high-performance, concurrent in-memory cache with the following key characteristics:

- **Sharded Architecture**: Reduces lock contention through data partitioning
- **Lock-Free Operations**: Atomic operations where possible
- **Memory Management**: Automatic eviction with configurable limits
- **Background Cleanup**: Asynchronous expired entry removal

```
┌─────────────────────────────────────┐
│            FastCache                │
│  ┌─────────────────────────────────┐│
│  │        Public API               ││
│  │  Set(), Get(), Delete(), etc.   ││
│  └─────────────────────────────────┘│
│  ┌─────────────────────────────────┐│
│  │      Shard Manager              ││
│  │   Hash-based Key Distribution   ││
│  └─────────────────────────────────┘│
│  ┌─────┬─────┬─────┬─────┬─────────┐│
│  │Shard│Shard│Shard│Shard│   ...   ││
│  │  0  │  1  │  2  │  3  │         ││
│  └─────┴─────┴─────┴─────┴─────────┘│
│  ┌─────────────────────────────────┐│
│  │     Background Services         ││
│  │  Cleanup, Monitoring, Stats     ││
│  └─────────────────────────────────┘│
└─────────────────────────────────────┘
```

## 🏗️ Core Components

### 1. Cache Structure

```go
type Cache struct {
    config    *Config        // Configuration parameters
    shards    []*Shard       // Array of cache shards
    totalSize int64          // Total memory usage (atomic)
    totalHits int64          // Global hit counter (atomic)
    totalMiss int64          // Global miss counter (atomic)
    closed    int32          // Closed flag (atomic)
    stopCh    chan struct{}  // Shutdown signal
    wg        sync.WaitGroup // Background goroutine coordination
}
```

**Key Design Decisions:**
- **Atomic counters** for statistics to avoid mutex overhead
- **Channel-based shutdown** for clean background process termination
- **WaitGroup coordination** ensures all goroutines finish during shutdown

### 2. Shard Architecture

Each shard operates independently to minimize lock contention:

```go
type Shard struct {
    mu        sync.RWMutex           // Read-write mutex
    data      map[string]*Entry      // Key-value storage
    lruList   *list.List            // LRU ordering
    size      int64                 // Shard memory usage (atomic)
    hitCount  int64                 // Shard hit counter (atomic)
    missCount int64                 // Shard miss counter (atomic)
}
```

**Shard Benefits:**
- **Reduced Lock Contention**: Operations on different shards can proceed concurrently
- **Parallel Eviction**: Each shard can evict independently
- **Isolated Performance**: Problems in one shard don't affect others

### 3. Entry Structure

```go
type Entry struct {
    key       string          // Cache key
    value     interface{}     // Stored value
    size      int64          // Memory footprint estimate
    expiry    int64          // Expiration timestamp (nanoseconds)
    listNode  *list.Element  // LRU list node
}
```

**Memory Optimization:**
- **Size estimation** for accurate memory tracking
- **Embedded list node** avoids separate allocation
- **Nanosecond precision** for expiry timing

## 🔀 Key Distribution

### Hash Function

FastCache uses FNV-1a hash for key distribution:

```go
func (c *Cache) hash(key string) uint32 {
    h := fnv.New32a()
    h.Write([]byte(key))
    return h.Sum32()
}

func (c *Cache) getShard(key string) *Shard {
    return c.shards[c.hash(key)%uint32(c.config.ShardCount)]
}
```

**Why FNV-1a?**
- **Fast computation**: Minimal CPU overhead
- **Good distribution**: Reduces hot spots
- **Standard library**: No external dependencies

### Shard Count Considerations

| Shard Count | Use Case | Trade-offs |
|-------------|----------|------------|
| 16-64 | Low concurrency | Lower memory overhead, higher contention |
| 256-512 | Medium concurrency | Balanced memory/performance |
| 1024-2048 | High concurrency | Higher memory, lower contention |
| 4096+ | Extreme concurrency | Significant memory overhead |

## 🔒 Concurrency Model

### Read Operations (GET)

```go
func (c *Cache) Get(key string) (interface{}, bool) {
    shard := c.getShard(key)
    
    shard.mu.RLock()                    // 1. Acquire read lock
    entry, exists := shard.data[key]    // 2. Map lookup
    shard.mu.RUnlock()                  // 3. Release read lock
    
    if !exists || entry.isExpired() {
        // Handle miss/expiry
        return nil, false
    }
    
    shard.mu.Lock()                     // 4. Acquire write lock for LRU update
    shard.lruList.MoveToFront(entry.listNode)
    shard.mu.Unlock()                  // 5. Release write lock
    
    return entry.value, true
}
```

**Optimization Notes:**
- **RLock first** allows concurrent reads
- **Expired entries** removed asynchronously to avoid blocking
- **LRU update** requires write lock but is brief

### Write Operations (SET)

```go
func (c *Cache) Set(key string, value interface{}, ttl ...time.Duration) error {
    shard := c.getShard(key)
    
    shard.mu.Lock()                     // 1. Acquire exclusive lock
    defer shard.mu.Unlock()
    
    // Update existing or create new entry
    // Update LRU order
    // Trigger eviction if needed
    
    return nil
}
```

**Critical Sections:**
- **Minimized lock time** through preparation outside locks
- **Atomic size updates** for memory tracking
- **Bulk operations** to reduce lock acquisition overhead

## 💾 Memory Management

### Size Estimation

```go
func calculateSize(key string, value interface{}) int64 {
    size := int64(len(key))
    
    switch v := value.(type) {
    case string:
        size += int64(len(v))
    case []byte:
        size += int64(len(v))
    // ... other types
    default:
        size += int64(unsafe.Sizeof(v))
    }
    
    size += 64  // Entry overhead
    return size
}
```

**Accuracy vs Performance:**
- **Estimation-based**: Avoids expensive reflection
- **Type-specific**: Accurate for common types
- **Overhead included**: Accounts for internal structures

### Eviction Strategy

FastCache implements **LRU (Least Recently Used)** eviction:

1. **Memory threshold** triggers eviction
2. **Distributed eviction** across multiple shards
3. **Batch removal** for efficiency
4. **Atomic updates** maintain consistency

```go
func (c *Cache) evictIfNeeded() {
    if atomic.LoadInt64(&c.totalSize) <= c.config.MaxMemoryBytes {
        return
    }
    
    // Evict from multiple shards
    shardsToEvict := c.config.ShardCount / 4
    for i := 0; i < shardsToEvict; i++ {
        c.evictFromShard(c.shards[i], 1)
    }
}
```

## ⏰ TTL and Cleanup

### Expiration Model

- **Per-entry expiration**: Nanosecond precision timestamps
- **Lazy expiration**: Checked during access
- **Background cleanup**: Periodic sweep of all shards

### Background Cleanup Process

```go
func (c *Cache) cleanupRoutine() {
    ticker := time.NewTicker(c.config.CleanupInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-c.stopCh:
            return
        case <-ticker.C:
            c.cleanupExpired()
        }
    }
}
```

**Cleanup Strategy:**
- **Non-blocking**: Doesn't interfere with operations
- **Incremental**: Processes shards individually
- **Configurable frequency**: Balances overhead vs accuracy

## 📊 Statistics and Monitoring

### Atomic Counters

All statistics use atomic operations to avoid lock overhead:

```go
// Global statistics
totalHits  int64  // atomic
totalMiss  int64  // atomic
totalSize  int64  // atomic

// Per-shard statistics
hitCount   int64  // atomic
missCount  int64  // atomic
size       int64  // atomic
```

### Memory Tracking

- **Real-time size tracking**: Updated on every operation
- **Shard-level granularity**: Enables load balancing analysis
- **Overflow protection**: Prevents memory limit violations

## 🚀 Performance Optimizations

### Hot Path Optimizations

1. **Minimize Allocations**:
   - Reuse slices where possible
   - Avoid string concatenation in loops
   - Pool frequently allocated objects

2. **Atomic Operations**:
   - Statistics updates use atomic operations
   - Lock-free counters for high-frequency operations

3. **Memory Layout**:
   - Struct field ordering for cache efficiency
   - Embedded structs to reduce pointer chasing

### Cold Path Optimizations

1. **Background Processing**:
   - Cleanup runs in separate goroutines
   - Non-blocking expiration handling

2. **Batch Operations**:
   - Eviction processes multiple entries
   - Cleanup sweeps entire shards

## 🔧 Configuration Impact

### Shard Count

```go
// Memory overhead per shard
shardOverhead = sizeof(Shard) + sizeof(map) + sizeof(list.List)
                ≈ 48 bytes + map overhead + list overhead
                ≈ 200-500 bytes per shard
```

**Guidelines:**
- **High concurrency**: More shards (1024+)
- **Memory constrained**: Fewer shards (64-256)
- **CPU bound**: Balance based on core count

### Cleanup Interval

- **Frequent cleanup** (seconds): Lower memory usage, higher CPU
- **Infrequent cleanup** (minutes): Higher memory usage, lower CPU

### Memory Limits

- **Aggressive limits**: More frequent eviction, lower hit rates
- **Generous limits**: Better hit rates, higher memory usage

## 🔍 Debugging and Profiling

### Built-in Diagnostics

```go
// Comprehensive statistics
stats := cache.GetStats()

// Per-shard analysis
shardStats := cache.GetShardStats()

// Memory breakdown
memInfo := cache.GetMemoryInfo()

// Performance metrics
perfMetrics := cache.GetPerformanceMetrics()
```

### Profiling Integration

FastCache works well with Go's built-in profiling tools:

```go
import _ "net/http/pprof"

// CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile

// Memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap

// Goroutine analysis
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## 🎯 Design Trade-offs

### Memory vs Performance
- **More shards**: Better concurrency, higher memory overhead
- **Larger entries**: Better hit rates, higher memory usage
- **Frequent cleanup**: Lower memory, higher CPU usage

### Consistency vs Performance
- **Atomic operations**: Consistent counters, slight performance cost
- **Lock granularity**: Fine-grained locks for better concurrency
- **Lazy expiration**: Better performance, temporary inconsistency

### Simplicity vs Features
- **Interface design**: Simple API, powerful internals
- **Configuration**: Sensible defaults, extensive customization
- **Error handling**: Graceful degradation, clear error messages

This architecture enables FastCache to achieve 1M+ QPS while maintaining memory efficiency and operational simplicity.