package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/nayan9229/fastcache"
)

// User represents a user entity
type User struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	LastSeen time.Time `json:"last_seen"`
}

// Product represents a product entity
type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Category    string  `json:"category"`
}

// APIServer represents our API server with caching
type APIServer struct {
	cache *fastcache.Cache
}

// NewAPIServer creates a new API server instance
func NewAPIServer() *APIServer {
	// Configure cache for API server workload
	config := &fastcache.Config{
		MaxMemoryBytes:  256 * 1024 * 1024, // 256MB for API cache
		ShardCount:      512,               // Good balance for web workload
		DefaultTTL:      15 * time.Minute,  // 15 minutes default cache
		CleanupInterval: 2 * time.Minute,   // Cleanup every 2 minutes
	}

	return &APIServer{
		cache: fastcache.New(config),
	}
}

// Close gracefully shuts down the server
func (s *APIServer) Close() {
	if err := s.cache.Close(); err != nil {
		log.Printf("Error closing cache: %v", err)
	}
}

// Simulate database operations with artificial latency
func (s *APIServer) fetchUserFromDB(userID int) (*User, error) {
	// Simulate database latency
	time.Sleep(10 * time.Millisecond)

	// Simulate user not found
	if userID <= 0 || userID > 1000 {
		return nil, fmt.Errorf("user not found")
	}

	// Return mock user data
	return &User{
		ID:       userID,
		Name:     fmt.Sprintf("User %d", userID),
		Email:    fmt.Sprintf("user%d@example.com", userID),
		Role:     []string{"user", "admin", "moderator"}[userID%3],
		LastSeen: time.Now().Add(-time.Duration(rand.Intn(3600)) * time.Second),
	}, nil
}

func (s *APIServer) fetchProductFromDB(productID int) (*Product, error) {
	// Simulate database latency
	time.Sleep(15 * time.Millisecond)

	if productID <= 0 || productID > 500 {
		return nil, fmt.Errorf("product not found")
	}

	categories := []string{"Electronics", "Books", "Clothing", "Home", "Sports"}
	names := []string{"Laptop", "Phone", "Tablet", "Book", "Shirt", "Pants", "Chair", "Table"}

	return &Product{
		ID:          productID,
		Name:        fmt.Sprintf("%s %d", names[productID%len(names)], productID),
		Description: fmt.Sprintf("Description for product %d", productID),
		Price:       float64(10 + rand.Intn(1000)),
		Stock:       rand.Intn(100),
		Category:    categories[productID%len(categories)],
	}, nil
}

// getUserHandler handles GET /api/users/{id}
func (s *APIServer) getUserHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Extract user ID from URL
	userIDStr := r.URL.Path[len("/api/users/"):]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Try cache first
	cacheKey := fmt.Sprintf("user:%d", userID)
	if cachedUser, exists := s.cache.Get(cacheKey); exists {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Header().Set("X-Response-Time", time.Since(start).String())
		json.NewEncoder(w).Encode(cachedUser)
		return
	}

	// Cache miss - fetch from database
	user, err := s.fetchUserFromDB(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Cache the result
	s.cache.Set(cacheKey, user, 10*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Header().Set("X-Response-Time", time.Since(start).String())
	json.NewEncoder(w).Encode(user)
}

// getProductHandler handles GET /api/products/{id}
func (s *APIServer) getProductHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	productIDStr := r.URL.Path[len("/api/products/"):]
	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	// Try cache first
	cacheKey := fmt.Sprintf("product:%d", productID)
	if cachedProduct, exists := s.cache.Get(cacheKey); exists {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Header().Set("X-Response-Time", time.Since(start).String())
		json.NewEncoder(w).Encode(cachedProduct)
		return
	}

	// Cache miss - fetch from database
	product, err := s.fetchProductFromDB(productID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Cache the result with longer TTL for products (they change less frequently)
	s.cache.Set(cacheKey, product, 30*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Header().Set("X-Response-Time", time.Since(start).String())
	json.NewEncoder(w).Encode(product)
}

// updateUserHandler handles PUT /api/users/{id}
func (s *APIServer) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Path[len("/api/users/"):]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Simulate database update
	time.Sleep(20 * time.Millisecond)

	// Invalidate cache after update
	cacheKey := fmt.Sprintf("user:%d", userID)
	s.cache.Delete(cacheKey)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "User updated successfully",
		"user_id": userID,
	})
}

// getCacheStatsHandler handles GET /api/cache/stats
func (s *APIServer) getCacheStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := s.cache.GetStats()
	memInfo := s.cache.GetMemoryInfo()
	perfMetrics := s.cache.GetPerformanceMetrics()

	response := map[string]interface{}{
		"stats":       stats,
		"memory_info": memInfo,
		"performance": perfMetrics,
		"timestamp":   time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// clearCacheHandler handles POST /api/cache/clear
func (s *APIServer) clearCacheHandler(w http.ResponseWriter, r *http.Request) {
	s.cache.Clear()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Cache cleared successfully",
	})
}

// healthHandler handles GET /health
func (s *APIServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	stats := s.cache.GetStats()
	memInfo := s.cache.GetMemoryInfo()

	health := map[string]interface{}{
		"status":        "healthy",
		"timestamp":     time.Now(),
		"cache_entries": stats.TotalEntries,
		"memory_usage":  memInfo.Percent,
		"hit_ratio":     stats.HitRatio * 100,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// loggingMiddleware logs all requests
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		duration := time.Since(start)
		log.Printf("%s %s - %v", r.Method, r.URL.Path, duration)
	}
}

func main() {
	log.Println("Starting FastCache API Server Example...")

	// Create API server with cache
	server := NewAPIServer()
	defer server.Close()

	// Setup routes
	http.HandleFunc("/api/users/", loggingMiddleware(server.getUserHandler))
	http.HandleFunc("/api/products/", loggingMiddleware(server.getProductHandler))
	http.HandleFunc("/api/cache/stats", loggingMiddleware(server.getCacheStatsHandler))
	http.HandleFunc("/api/cache/clear", loggingMiddleware(server.clearCacheHandler))
	http.HandleFunc("/health", loggingMiddleware(server.healthHandler))

	// Setup user update route (requires exact match)
	http.HandleFunc("/api/users/update/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			// Extract ID from /api/users/update/{id}
			userIDStr := r.URL.Path[len("/api/users/update/"):]
			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				http.Error(w, "Invalid user ID", http.StatusBadRequest)
				return
			}

			// Create a new request with the correct path for updateUserHandler
			r.URL.Path = fmt.Sprintf("/api/users/%d", userID)
			loggingMiddleware(server.updateUserHandler)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Start background monitoring
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := server.cache.GetStats()
			log.Printf("Cache Stats: Entries=%d, Memory=%s, Hit Ratio=%.2f%%",
				stats.TotalEntries, stats.MemoryUsage, stats.HitRatio*100)
		}
	}()

	// Warm up cache with some data
	go func() {
		log.Println("Warming up cache...")
		for i := 1; i <= 100; i++ {
			user, _ := server.fetchUserFromDB(i)
			if user != nil {
				server.cache.Set(fmt.Sprintf("user:%d", i), user)
			}

			product, _ := server.fetchProductFromDB(i)
			if product != nil {
				server.cache.Set(fmt.Sprintf("product:%d", i), product)
			}
		}
		log.Println("Cache warm-up completed")
	}()

	// Print usage information
	fmt.Println("\n=== FastCache API Server Running ===")
	fmt.Println("Server: http://localhost:8080")
	fmt.Println("\nAvailable endpoints:")
	fmt.Println("GET  /api/users/{id}      - Get user by ID")
	fmt.Println("GET  /api/products/{id}   - Get product by ID")
	fmt.Println("PUT  /api/users/update/{id} - Update user (invalidates cache)")
	fmt.Println("GET  /api/cache/stats     - Get cache statistics")
	fmt.Println("POST /api/cache/clear     - Clear cache")
	fmt.Println("GET  /health              - Health check")
	fmt.Println("\nExample requests:")
	fmt.Println("curl http://localhost:8080/api/users/123")
	fmt.Println("curl http://localhost:8080/api/products/456")
	fmt.Println("curl http://localhost:8080/api/cache/stats")
	fmt.Println("curl -X PUT http://localhost:8080/api/users/update/123 -d '{\"name\":\"Updated Name\"}'")
	fmt.Println("\nPress Ctrl+C to stop")

	// Start server
	log.Fatal(http.ListenAndServe(":8080", nil))
}
