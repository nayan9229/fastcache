package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/nayan9229/fastcache"
)

// MonitoringServer provides cache monitoring capabilities
type MonitoringServer struct {
	cache      *fastcache.Cache
	startTime  time.Time
	alertRules []AlertRule
}

// AlertRule defines monitoring alert conditions
type AlertRule struct {
	Name      string
	Condition func(*fastcache.Stats) bool
	Message   string
	Triggered bool
	LastAlert time.Time
}

// MetricsSnapshot captures point-in-time metrics
type MetricsSnapshot struct {
	Timestamp     time.Time                     `json:"timestamp"`
	Stats         *fastcache.Stats              `json:"stats"`
	MemoryInfo    *fastcache.MemoryInfo         `json:"memory_info"`
	Performance   *fastcache.PerformanceMetrics `json:"performance"`
	SystemMetrics *SystemMetrics                `json:"system_metrics"`
	ShardStats    []fastcache.ShardStats        `json:"shard_stats"`
	Alerts        []string                      `json:"alerts"`
}

// SystemMetrics captures system-level metrics
type SystemMetrics struct {
	NumGoroutines int              `json:"num_goroutines"`
	MemStats      runtime.MemStats `json:"mem_stats"`
	// GCStats       runtime.GCStats  `json:"gc_stats"`
}

func main() {
	fmt.Println("=== FastCache Monitoring & Metrics Example ===")

	// Create monitoring server
	server := NewMonitoringServer()
	defer server.Close()

	// Start background workload simulation
	go server.simulateWorkload()

	// Start metrics collection
	go server.startMetricsCollection()

	// Start alert monitoring
	go server.startAlertMonitoring()

	// Start HTTP metrics endpoint
	go server.startHTTPServer()

	// Show real-time dashboard
	server.runDashboard()
}

func NewMonitoringServer() *MonitoringServer {
	config := &fastcache.Config{
		MaxMemoryBytes:  128 * 1024 * 1024, // 128MB for demo
		ShardCount:      256,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 30 * time.Second,
	}

	cache := fastcache.New(config)

	// Define alert rules
	alertRules := []AlertRule{
		{
			Name: "High Memory Usage",
			Condition: func(stats *fastcache.Stats) bool {
				return stats.MemoryPercent > 80.0
			},
			Message: "Cache memory usage exceeded 80%",
		},
		{
			Name: "Low Hit Ratio",
			Condition: func(stats *fastcache.Stats) bool {
				return stats.HitCount+stats.MissCount > 1000 && stats.HitRatio < 0.7
			},
			Message: "Cache hit ratio dropped below 70%",
		},
		{
			Name: "High Operation Count",
			Condition: func(stats *fastcache.Stats) bool {
				return stats.HitCount+stats.MissCount > 100000
			},
			Message: "Cache operation count exceeded 100K",
		},
	}

	return &MonitoringServer{
		cache:      cache,
		startTime:  time.Now(),
		alertRules: alertRules,
	}
}

func (m *MonitoringServer) Close() {
	m.cache.Close()
}

func (m *MonitoringServer) simulateWorkload() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		// Simulate different types of workloads
		r := rand.Float32()

		if r < 0.3 {
			// Write operation
			key := fmt.Sprintf("workload_key_%d", rand.Intn(1000))
			value := fmt.Sprintf("workload_value_%d_%d", rand.Intn(1000), time.Now().Unix())
			m.cache.Set(key, value)
		} else {
			// Read operation
			key := fmt.Sprintf("workload_key_%d", rand.Intn(1200)) // Some misses
			m.cache.Get(key)
		}

		// Occasionally delete some keys
		if rand.Float32() < 0.05 {
			key := fmt.Sprintf("workload_key_%d", rand.Intn(1000))
			m.cache.Delete(key)
		}
	}
}

func (m *MonitoringServer) startMetricsCollection() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		snapshot := m.captureMetricsSnapshot()
		m.logMetrics(snapshot)
	}
}

func (m *MonitoringServer) startAlertMonitoring() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := m.cache.GetStats()

		for i := range m.alertRules {
			rule := &m.alertRules[i]

			if rule.Condition(stats) {
				if !rule.Triggered || time.Since(rule.LastAlert) > 5*time.Minute {
					log.Printf("🚨 ALERT: %s - %s", rule.Name, rule.Message)
					rule.Triggered = true
					rule.LastAlert = time.Now()
				}
			} else {
				if rule.Triggered {
					log.Printf("✅ RESOLVED: %s", rule.Name)
					rule.Triggered = false
				}
			}
		}
	}
}

func (m *MonitoringServer) startHTTPServer() {
	http.HandleFunc("/metrics", m.metricsHandler)
	http.HandleFunc("/metrics/json", m.metricsJSONHandler)
	http.HandleFunc("/metrics/prometheus", m.prometheusHandler)
	http.HandleFunc("/health", m.healthHandler)
	http.HandleFunc("/alerts", m.alertsHandler)

	fmt.Println("Metrics HTTP server started on :8090")
	fmt.Println("Available endpoints:")
	fmt.Println("  http://localhost:8090/metrics - Human readable metrics")
	fmt.Println("  http://localhost:8090/metrics/json - JSON metrics")
	fmt.Println("  http://localhost:8090/metrics/prometheus - Prometheus format")
	fmt.Println("  http://localhost:8090/health - Health check")
	fmt.Println("  http://localhost:8090/alerts - Active alerts")

	log.Fatal(http.ListenAndServe(":8090", nil))
}

func (m *MonitoringServer) runDashboard() {
	fmt.Println("Starting real-time dashboard (Press Ctrl+C to stop)...")
	fmt.Println("=")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.displayDashboard()
	}
}

func (m *MonitoringServer) displayDashboard() {
	// Clear screen (works on most terminals)
	fmt.Print("\033[2J\033[H")

	snapshot := m.captureMetricsSnapshot()

	fmt.Println("🚀 FastCache Real-Time Dashboard")
	fmt.Println("=")
	fmt.Printf("Uptime: %v | Last Updated: %s\n",
		time.Since(m.startTime).Round(time.Second),
		snapshot.Timestamp.Format("15:04:05"))
	fmt.Println()

	// Cache Statistics
	fmt.Println("📊 Cache Statistics:")
	fmt.Printf("  Entries: %d | Memory: %s (%.1f%% of %.1f MB)\n",
		snapshot.Stats.TotalEntries,
		snapshot.Stats.MemoryUsage,
		snapshot.MemoryInfo.Percent,
		float64(snapshot.MemoryInfo.Max)/1024/1024)

	fmt.Printf("  Hit Ratio: %.2f%% | Hits: %d | Misses: %d\n",
		snapshot.Stats.HitRatio*100,
		snapshot.Stats.HitCount,
		snapshot.Stats.MissCount)

	fmt.Printf("  Shards: %d | Operations: %d\n",
		snapshot.Stats.ShardCount,
		snapshot.Performance.TotalOperations)
	fmt.Println()

	// Performance Metrics
	fmt.Println("⚡ Performance:")
	fmt.Printf("  Avg Shard Load: %.1f | Max: %d | Min: %d\n",
		snapshot.Performance.AvgShardLoad,
		snapshot.Performance.MaxShardLoad,
		snapshot.Performance.MinShardLoad)

	fmt.Printf("  Load Balance: %.2f | Hit Rate: %.2f%% | Miss Rate: %.2f%%\n",
		snapshot.Performance.LoadBalance,
		snapshot.Performance.HitRate*100,
		snapshot.Performance.MissRate*100)
	fmt.Println()

	// System Metrics
	fmt.Println("🖥️  System:")
	fmt.Printf("  Goroutines: %d | Heap: %.1f MB | GC Runs: %d\n",
		snapshot.SystemMetrics.NumGoroutines,
		float64(snapshot.SystemMetrics.MemStats.HeapAlloc)/1024/1024,
		snapshot.SystemMetrics.MemStats.NumGC)

	fmt.Printf("  Next GC: %.1f MB | Last GC: %v ago\n",
		float64(snapshot.SystemMetrics.MemStats.NextGC)/1024/1024,
		time.Since(time.Unix(0, int64(snapshot.SystemMetrics.MemStats.LastGC))).Round(time.Second))
	fmt.Println()

	// Memory Distribution (top 5 shards)
	fmt.Println("💾 Memory Distribution (Top 5 Shards):")
	for _, shard := range snapshot.ShardStats[:min(5, len(snapshot.ShardStats))] {
		fmt.Printf("  Shard %d: %d entries, %s, %.2f%% hit ratio\n",
			shard.ShardID,
			shard.EntryCount,
			shard.MemoryUsage,
			shard.HitRatio*100)
	}
	fmt.Println()

	// Active Alerts
	if len(snapshot.Alerts) > 0 {
		fmt.Println("🚨 Active Alerts:")
		for _, alert := range snapshot.Alerts {
			fmt.Printf("  • %s\n", alert)
		}
		fmt.Println()
	}

	// Progress bars for visual representation
	m.displayProgressBars(snapshot)
}

func (m *MonitoringServer) displayProgressBars(snapshot *MetricsSnapshot) {
	fmt.Println("📈 Visual Indicators:")

	// Memory usage bar
	memPercent := snapshot.MemoryInfo.Percent
	fmt.Printf("  Memory:    [%s] %.1f%%\n",
		createProgressBar(memPercent, 50), memPercent)

	// Hit ratio bar
	hitPercent := snapshot.Stats.HitRatio * 100
	fmt.Printf("  Hit Ratio: [%s] %.1f%%\n",
		createProgressBar(hitPercent, 50), hitPercent)

	// Load balance indicator (lower is better)
	// loadBalance := min(snapshot.Performance.LoadBalance/100*100, 100) // Normalize
	// fmt.Printf("  Balance:   [%s] %.1f\n",
	// 	createProgressBar(100-loadBalance, 50), snapshot.Performance.LoadBalance)
}

func createProgressBar(percent float64, width int) string {
	filled := int(percent * float64(width) / 100)
	bar := ""

	for i := 0; i < width; i++ {
		if i < filled {
			if percent > 80 {
				bar += "█" // Red zone
			} else if percent > 60 {
				bar += "▓" // Yellow zone
			} else {
				bar += "▒" // Green zone
			}
		} else {
			bar += "░"
		}
	}

	return bar
}

func (m *MonitoringServer) captureMetricsSnapshot() *MetricsSnapshot {
	stats := m.cache.GetStats()
	memInfo := m.cache.GetMemoryInfo()
	performance := m.cache.GetPerformanceMetrics()
	shardStats := m.cache.GetShardStats()

	// Capture system metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// var gcStats runtime.GCStats
	// runtime.ReadGCStats(&gcStats)

	systemMetrics := &SystemMetrics{
		NumGoroutines: runtime.NumGoroutine(),
		MemStats:      memStats,
		// GCStats:       gcStats,
	}

	// Collect active alerts
	var activeAlerts []string
	for _, rule := range m.alertRules {
		if rule.Triggered {
			activeAlerts = append(activeAlerts, rule.Message)
		}
	}

	return &MetricsSnapshot{
		Timestamp:     time.Now(),
		Stats:         stats,
		MemoryInfo:    memInfo,
		Performance:   performance,
		SystemMetrics: systemMetrics,
		ShardStats:    shardStats,
		Alerts:        activeAlerts,
	}
}

func (m *MonitoringServer) logMetrics(snapshot *MetricsSnapshot) {
	log.Printf("Metrics: entries=%d, memory=%.1f%%, hit_ratio=%.2f%%, ops=%d, goroutines=%d",
		snapshot.Stats.TotalEntries,
		snapshot.MemoryInfo.Percent,
		snapshot.Stats.HitRatio*100,
		snapshot.Performance.TotalOperations,
		snapshot.SystemMetrics.NumGoroutines)
}

// HTTP Handlers

func (m *MonitoringServer) metricsHandler(w http.ResponseWriter, r *http.Request) {
	snapshot := m.captureMetricsSnapshot()

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "FastCache Metrics\n")
	fmt.Fprintf(w, "================\n")
	fmt.Fprintf(w, "Timestamp: %s\n", snapshot.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(w, "Uptime: %v\n", time.Since(m.startTime))
	fmt.Fprintf(w, "\nCache Statistics:\n")
	fmt.Fprintf(w, "  Total Entries: %d\n", snapshot.Stats.TotalEntries)
	fmt.Fprintf(w, "  Memory Usage: %s (%.1f%%)\n", snapshot.Stats.MemoryUsage, snapshot.MemoryInfo.Percent)
	fmt.Fprintf(w, "  Hit Count: %d\n", snapshot.Stats.HitCount)
	fmt.Fprintf(w, "  Miss Count: %d\n", snapshot.Stats.MissCount)
	fmt.Fprintf(w, "  Hit Ratio: %.2f%%\n", snapshot.Stats.HitRatio*100)
}

func (m *MonitoringServer) metricsJSONHandler(w http.ResponseWriter, r *http.Request) {
	snapshot := m.captureMetricsSnapshot()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

func (m *MonitoringServer) prometheusHandler(w http.ResponseWriter, r *http.Request) {
	snapshot := m.captureMetricsSnapshot()

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "# HELP fastcache_entries_total Total number of cache entries\n")
	fmt.Fprintf(w, "# TYPE fastcache_entries_total gauge\n")
	fmt.Fprintf(w, "fastcache_entries_total %d\n", snapshot.Stats.TotalEntries)

	fmt.Fprintf(w, "# HELP fastcache_memory_bytes Memory usage in bytes\n")
	fmt.Fprintf(w, "# TYPE fastcache_memory_bytes gauge\n")
	fmt.Fprintf(w, "fastcache_memory_bytes %d\n", snapshot.Stats.TotalSize)

	fmt.Fprintf(w, "# HELP fastcache_hit_ratio Cache hit ratio\n")
	fmt.Fprintf(w, "# TYPE fastcache_hit_ratio gauge\n")
	fmt.Fprintf(w, "fastcache_hit_ratio %.4f\n", snapshot.Stats.HitRatio)

	fmt.Fprintf(w, "# HELP fastcache_operations_total Total cache operations\n")
	fmt.Fprintf(w, "# TYPE fastcache_operations_total counter\n")
	fmt.Fprintf(w, "fastcache_operations_total{type=\"hit\"} %d\n", snapshot.Stats.HitCount)
	fmt.Fprintf(w, "fastcache_operations_total{type=\"miss\"} %d\n", snapshot.Stats.MissCount)
}

func (m *MonitoringServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	stats := m.cache.GetStats()

	health := map[string]interface{}{
		"status":         "healthy",
		"timestamp":      time.Now(),
		"uptime":         time.Since(m.startTime).String(),
		"memory_percent": stats.MemoryPercent,
		"hit_ratio":      stats.HitRatio,
		"entries":        stats.TotalEntries,
	}

	// Determine health status
	if stats.MemoryPercent > 95 {
		health["status"] = "critical"
	} else if stats.MemoryPercent > 80 || stats.HitRatio < 0.5 {
		health["status"] = "warning"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (m *MonitoringServer) alertsHandler(w http.ResponseWriter, r *http.Request) {
	var activeAlerts []map[string]interface{}

	for _, rule := range m.alertRules {
		if rule.Triggered {
			activeAlerts = append(activeAlerts, map[string]interface{}{
				"name":      rule.Name,
				"message":   rule.Message,
				"triggered": rule.LastAlert,
			})
		}
	}

	response := map[string]interface{}{
		"alerts":    activeAlerts,
		"count":     len(activeAlerts),
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
