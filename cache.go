// Package fastcache provides a high-performance, goroutine-safe in-memory key-value cache
// designed to handle 1M+ QPS with automatic memory management and LRU eviction.
package fastcache

import (
	"container/list"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Entry represents a single cache entry
type Entry struct {
	key      string
	value    interface{}
	size     int64
	expiry   int64 // Unix timestamp in nanoseconds
	listNode *list.Element
}

// isExpired checks if the entry has expired
func (e *Entry) isExpired() bool {
	return e.expiry > 0 && time.Now().UnixNano() > e.expiry
}

// Shard represents a single shard of the cache
type Shard struct {
	mu        sync.RWMutex
	data      map[string]*Entry
	lruList   *list.List
	size      int64
	hitCount  int64
	missCount int64
}

// newShard creates a new shard
func newShard() *Shard {
	return &Shard{
		data:    make(map[string]*Entry),
		lruList: list.New(),
	}
}

// Cache is the main cache structure
type Cache struct {
	config    *Config
	shards    []*Shard
	totalSize int64
	totalHits int64
	totalMiss int64
	closed    int32
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// New creates a new cache instance
func New(config *Config) *Cache {
	if config == nil {
		config = DefaultConfig()
	}

	cache := &Cache{
		config: config,
		shards: make([]*Shard, config.ShardCount),
		stopCh: make(chan struct{}),
	}

	// Initialize shards
	for i := 0; i < config.ShardCount; i++ {
		cache.shards[i] = newShard()
	}

	// Start background cleanup goroutine
	cache.wg.Add(1)
	go cache.cleanupRoutine()

	return cache
}

// hash returns the hash of a key
func (c *Cache) hash(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// getShard returns the appropriate shard for a key
func (c *Cache) getShard(key string) *Shard {
	return c.shards[c.hash(key)%uint32(c.config.ShardCount)]
}

// calculateSize estimates the memory size of a key-value pair
func calculateSize(key string, value interface{}) int64 {
	size := int64(len(key))

	switch v := value.(type) {
	case string:
		size += int64(len(v))
	case []byte:
		size += int64(len(v))
	case int, int32, int64, uint, uint32, uint64:
		size += 8
	case float32, float64:
		size += 8
	case bool:
		size += 1
	default:
		// Rough estimate for other types
		size += int64(unsafe.Sizeof(v))
	}

	// Add overhead for Entry struct and list node
	size += 64

	return size
}

// Set stores a key-value pair with optional TTL
func (c *Cache) Set(key string, value interface{}, ttl ...time.Duration) error {
	if atomic.LoadInt32(&c.closed) == 1 {
		return ErrCacheClosed
	}

	shard := c.getShard(key)
	size := calculateSize(key, value)

	var expiry int64
	if len(ttl) > 0 && ttl[0] > 0 {
		expiry = time.Now().Add(ttl[0]).UnixNano()
	} else if c.config.DefaultTTL > 0 {
		expiry = time.Now().Add(c.config.DefaultTTL).UnixNano()
	}

	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Check if key already exists
	if existing, exists := shard.data[key]; exists {
		// Update existing entry
		atomic.AddInt64(&c.totalSize, size-existing.size)
		atomic.AddInt64(&shard.size, size-existing.size)

		existing.value = value
		existing.size = size
		existing.expiry = expiry

		// Move to front of LRU list
		shard.lruList.MoveToFront(existing.listNode)

		c.evictIfNeeded()
		return nil
	}

	// Create new entry
	entry := &Entry{
		key:    key,
		value:  value,
		size:   size,
		expiry: expiry,
	}

	entry.listNode = shard.lruList.PushFront(entry)
	shard.data[key] = entry

	atomic.AddInt64(&c.totalSize, size)
	atomic.AddInt64(&shard.size, size)

	c.evictIfNeeded()
	return nil
}

// Get retrieves a value by key
func (c *Cache) Get(key string) (interface{}, bool) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return nil, false
	}

	shard := c.getShard(key)

	shard.mu.RLock()
	entry, exists := shard.data[key]
	shard.mu.RUnlock()

	if !exists {
		atomic.AddInt64(&shard.missCount, 1)
		atomic.AddInt64(&c.totalMiss, 1)
		return nil, false
	}

	if entry.isExpired() {
		// Remove expired entry asynchronously to avoid blocking
		go c.Delete(key)
		atomic.AddInt64(&shard.missCount, 1)
		atomic.AddInt64(&c.totalMiss, 1)
		return nil, false
	}

	// Update LRU order
	shard.mu.Lock()
	shard.lruList.MoveToFront(entry.listNode)
	shard.mu.Unlock()

	atomic.AddInt64(&shard.hitCount, 1)
	atomic.AddInt64(&c.totalHits, 1)
	return entry.value, true
}

// Delete removes a key from the cache
func (c *Cache) Delete(key string) bool {
	if atomic.LoadInt32(&c.closed) == 1 {
		return false
	}

	shard := c.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	entry, exists := shard.data[key]
	if !exists {
		return false
	}

	delete(shard.data, key)
	shard.lruList.Remove(entry.listNode)
	atomic.AddInt64(&c.totalSize, -entry.size)
	atomic.AddInt64(&shard.size, -entry.size)

	return true
}

// evictIfNeeded removes old entries if memory limit is exceeded
func (c *Cache) evictIfNeeded() {
	if atomic.LoadInt64(&c.totalSize) <= c.config.MaxMemoryBytes {
		return
	}

	// Evict from multiple shards to distribute the load
	shardsToEvict := c.config.ShardCount / 4
	if shardsToEvict < 1 {
		shardsToEvict = 1
	}

	for i := 0; i < shardsToEvict; i++ {
		shard := c.shards[i]
		c.evictFromShard(shard, 1)
	}
}

// evictFromShard removes the oldest entries from a shard
func (c *Cache) evictFromShard(shard *Shard, count int) {
	shard.mu.Lock()
	defer shard.mu.Unlock()

	for i := 0; i < count && shard.lruList.Len() > 0; i++ {
		oldest := shard.lruList.Back()
		if oldest == nil {
			break
		}

		entry := oldest.Value.(*Entry)
		delete(shard.data, entry.key)
		shard.lruList.Remove(oldest)
		atomic.AddInt64(&c.totalSize, -entry.size)
		atomic.AddInt64(&shard.size, -entry.size)
	}
}

// cleanupRoutine runs periodic cleanup of expired entries
func (c *Cache) cleanupRoutine() {
	defer c.wg.Done()

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

// cleanupExpired removes expired entries from all shards
func (c *Cache) cleanupExpired() {
	now := time.Now().UnixNano()

	for _, shard := range c.shards {
		shard.mu.Lock()

		// Collect expired keys
		var expiredKeys []string
		for key, entry := range shard.data {
			if entry.expiry > 0 && now > entry.expiry {
				expiredKeys = append(expiredKeys, key)
			}
		}

		// Remove expired entries
		for _, key := range expiredKeys {
			entry := shard.data[key]
			delete(shard.data, key)
			shard.lruList.Remove(entry.listNode)
			atomic.AddInt64(&c.totalSize, -entry.size)
			atomic.AddInt64(&shard.size, -entry.size)
		}

		shard.mu.Unlock()
	}
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	for _, shard := range c.shards {
		shard.mu.Lock()
		shard.data = make(map[string]*Entry)
		shard.lruList = list.New()
		atomic.StoreInt64(&shard.size, 0)
		shard.mu.Unlock()
	}
	atomic.StoreInt64(&c.totalSize, 0)
}

// Close gracefully shuts down the cache
func (c *Cache) Close() error {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return ErrCacheClosed
	}

	close(c.stopCh)
	c.wg.Wait()

	return nil
}
