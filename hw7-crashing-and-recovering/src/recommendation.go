package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// ── Recommendation Service ─────────────────────────────────────────────────────
// A simulated slow external dependency that demonstrates cascading failure
// Runs on port 8081

type RecommendationResponse struct {
	Query            string                   `json:"query"`
	Recommendations  []map[string]interface{} `json:"recommendations"`
	LatencyMs        int64                    `json:"latency_ms"`
	Status           string                   `json:"status"`
	CurrentLoad      int                      `json:"current_load"`
	MaxConcurrent    int                      `json:"max_concurrent"`
}

type RecommendationEngine struct {
	maxConcurrentRequests int
	currentRequests       int32
	latencyMs             int
}

var recEngine *RecommendationEngine

// Helper function to parse environment variables as integers
func parseEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	if parsed < 0 {
		return fallback
	}

	return parsed
}

// ── HTTP Handlers for Recommendation Service ──────────────────────────────────

// recommendationsHandler handles product recommendation requests
// Simulates a slow ML/database operation
func recommendationsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, `{"error":"missing query param q"}`, http.StatusBadRequest)
		return
	}

	// Track concurrent requests
	current := atomic.AddInt32(&recEngine.currentRequests, 1)
	defer atomic.AddInt32(&recEngine.currentRequests, -1)

	// Check if overloaded
	if int(current) > recEngine.maxConcurrentRequests {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":            "recommendation service overloaded",
			"current_requests": current,
			"max_concurrent":   recEngine.maxConcurrentRequests,
		})
		return
	}

	// Simulate expensive operation: ML model inference, DB aggregation, etc.
	time.Sleep(time.Duration(recEngine.latencyMs) * time.Millisecond)

	// Build dummy recommendations
	elapsed := time.Since(start)
	recommendations := []map[string]interface{}{
		{
			"product_id": 42,
			"name":       fmt.Sprintf("Recommended_%s_1", query),
			"score":      0.95,
			"reason":     "frequently purchased with this search",
		},
		{
			"product_id": 88,
			"name":       fmt.Sprintf("Recommended_%s_2", query),
			"score":      0.87,
			"reason":     "trending in this category",
		},
		{
			"product_id": 156,
			"name":       fmt.Sprintf("Recommended_%s_3", query),
			"score":      0.72,
			"reason":     "complements this purchase",
		},
	}

	resp := RecommendationResponse{
		Query:           query,
		Recommendations: recommendations,
		LatencyMs:       elapsed.Milliseconds(),
		Status:          "success",
		CurrentLoad:     int(atomic.LoadInt32(&recEngine.currentRequests)),
		MaxConcurrent:   recEngine.maxConcurrentRequests,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	log.Printf("[REC] Query=%s | Concurrent=%d/%d | Latency=%dms",
		query, int(current), recEngine.maxConcurrentRequests, elapsed.Milliseconds())
}

// recHealthHandler returns health status based on current load
func recHealthHandler(w http.ResponseWriter, r *http.Request) {
	current := atomic.LoadInt32(&recEngine.currentRequests)
	status := "ok"
	code := http.StatusOK

	if int(current) >= recEngine.maxConcurrentRequests {
		status = "overloaded"
		code = http.StatusServiceUnavailable
	}

	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    status,
		"load":      int(current),
		"max":       recEngine.maxConcurrentRequests,
		"timestamp": time.Now().Unix(),
	})
}

// recStatusHandler returns detailed status information
func recStatusHandler(w http.ResponseWriter, r *http.Request) {
	current := atomic.LoadInt32(&recEngine.currentRequests)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"current_requests": int(current),
		"max_concurrent":   recEngine.maxConcurrentRequests,
		"latency_ms":       recEngine.latencyMs,
		"load_percent":     (int(current) * 100) / recEngine.maxConcurrentRequests,
	})
}

// ── Server Startup ─────────────────────────────────────────────────────────────

// StartRecommendationServer starts the recommendation service on port 8081
func StartRecommendationServer() {
	latencyMs := parseEnvInt("REC_LATENCY_MS", 500)
	maxConcurrent := parseEnvInt("REC_MAX_CONCURRENT", 10)

	recEngine = &RecommendationEngine{
		maxConcurrentRequests: maxConcurrent,
		currentRequests:       0,
		latencyMs:             latencyMs,
	}

	http.HandleFunc("/recommendations", recommendationsHandler)
	http.HandleFunc("/rec_health", recHealthHandler)
	http.HandleFunc("/rec_status", recStatusHandler)

	go func() {
		log.Printf("[REC] Starting recommendation service on :8081 (latency=%dms, max_concurrent=%d)",
			latencyMs, maxConcurrent)

		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Fatalf("[REC] Failed to start: %v", err)
		}
	}()
}
