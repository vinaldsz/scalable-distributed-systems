package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ── Data Model ────────────────────────────────────────────────────────────────
type Product struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Brand       string `json:"brand"`
}

type SearchResponse struct {
	Products         []Product       `json:"products"`
	TotalFound       int             `json:"total_found"`
	Recommendations  []interface{}   `json:"recommendations"`
	SearchTimeMs     int64           `json:"search_time_ms"`
	RecommendationMs int64           `json:"recommendation_time_ms"`
	TotalTimeMs      int64           `json:"total_time_ms"`
	ActiveGoroutines int             `json:"active_goroutines"`
	Version          string          `json:"version"`
}

// ── In-memory store ───────────────────────────────────────────────────────────
var (
	store               sync.Map // key: int, value: Product
	activeGoroutines    int32    // Track goroutines for metrics
)

// ── Data generation ───────────────────────────────────────────────────────────
var (
	brands      = []string{"Alpha", "Beta", "Gamma", "Delta", "Echo", "Foxtrot"}
	categories  = []string{"Electronics", "Books", "Home", "Sports", "Clothing", "Toys"}
	adjectives  = []string{"Pro", "Ultra", "Lite", "Max", "Plus", "Smart"}
)

func generateProducts(n int) {
	for i := 1; i <= n; i++ {
		brand := brands[i%len(brands)]
		cat := categories[i%len(categories)]
		adj := adjectives[i%len(adjectives)]
		p := Product{
			ID:          i,
			Name:        fmt.Sprintf("Product %s %d", brand, i),
			Category:    cat,
			Description: fmt.Sprintf("%s %s item #%d", adj, cat, i),
			Brand:       brand,
		}
		store.Store(i, p)
	}
	log.Printf("[SEARCH] Generated %d products", n)
}

// ── Search logic ──────────────────────────────────────────────────────────────
const (
	totalProducts = 10_000
	checkLimit    = 100
	maxResults    = 20
)

func search(query string) ([]Product, int) {
	q := strings.ToLower(query)
	var results []Product
	totalFound := 0
	checked := 0

	for id := 1; id <= totalProducts; id++ {
		if checked >= checkLimit {
			break
		}

		val, ok := store.Load(id)
		if !ok {
			continue
		}

		p := val.(Product)
		if strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(strings.ToLower(p.Category), q) {
			totalFound++
			if len(results) < maxResults {
				results = append(results, p)
			}
		}
		checked++
	}

	return results, totalFound
}

// ── HTTP Clients ──────────────────────────────────────────────────────────────
type RecommendationClient struct {
	baseURL string
	timeout time.Duration
	version string  // "broken" or "fixed"
}

var recClient *RecommendationClient

// ❌ BROKEN VERSION: No timeout, no bulkhead, waits indefinitely
// FetchRecommendations calls the slow recommendation service
func (rc *RecommendationClient) FetchRecommendations(ctx context.Context, query string) ([]interface{}, int64, error) {
	// ❌ PROBLEM: No timeout context passed, no semaphore, unbounded goroutines
	start := time.Now()
	url := fmt.Sprintf("%s/recommendations?q=%s", rc.baseURL, query)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, time.Since(start).Milliseconds(), fmt.Errorf("recommendation service error: %d - %s",
			resp.StatusCode, string(body))
	}

	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)

	var recs []interface{}
	if recsData, ok := data["recommendations"].([]interface{}); ok {
		recs = recsData
	}

	return recs, time.Since(start).Milliseconds(), nil
}

// ❌ BROKEN: No protection for slow recommendation calls
// ── HTTP Handlers ─────────────────────────────────────────────────────────────
func searchHandlerBroken(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt32(&activeGoroutines, 1)
	defer atomic.AddInt32(&activeGoroutines, -1)

	start := time.Now()

	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, `{"error":"missing query param q"}`, http.StatusBadRequest)
		return
	}

	// Step 1: Quick search in memory
	searchStart := time.Now()
	products, total := search(q)
	searchTimeMs := time.Since(searchStart).Milliseconds()

	// Step 2: ❌ PROBLEM: Call slow recommendation service with NO PROTECTION
	// This blocks the goroutine for 500ms+ even if service is slow/overloaded
	recommendations := []interface{}{}

	ctx, cancel := context.WithCancel(context.Background())
	// Create a context with no timeout for broken version
	defer cancel()

	recs, recTimeMs, err := recClient.FetchRecommendations(ctx, q)
	if err == nil {
		recommendations = recs
	} else {
		log.Printf("[SEARCH] Recommendation error: %v", err)
	}

	totalTimeMs := time.Since(start).Milliseconds()

	resp := SearchResponse{
		Products:         products,
		TotalFound:       total,
		Recommendations:  recommendations,
		SearchTimeMs:     searchTimeMs,
		RecommendationMs: recTimeMs,
		TotalTimeMs:      totalTimeMs,
		ActiveGoroutines: int(atomic.LoadInt32(&activeGoroutines)),
		Version:          recClient.version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	log.Printf("[SEARCH] Query=%s | Search=%dms | Rec=%dms | Total=%dms | Goroutines=%d",
		q, searchTimeMs, recTimeMs, totalTimeMs, int(atomic.LoadInt32(&activeGoroutines)))
}

// ── BULKHEAD PATTERN: Fixed Version ──────────────────────────────────────────
// Limits concurrent calls to recommendation service so slow dependencies
// don't starve critical paths like /health

var (
	recSemaphore  chan struct{}    // Bounded semaphore for recommendations
	bucketDepth   = 5              // Max concurrent recommendation calls
	bucketTimeout = 100 * time.Millisecond  // Timeout for acquiring slot
)

func InitializeBulkhead() {
	recSemaphore = make(chan struct{}, bucketDepth)
	log.Printf("[BULKHEAD] Initialized with depth=%d, timeout=%dms", bucketDepth, bucketTimeout.Milliseconds())
}

// ✓ FIXED: Using bulkhead semaphore to limit concurrent calls
func searchHandlerFixed(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt32(&activeGoroutines, 1)
	defer atomic.AddInt32(&activeGoroutines, -1)

	start := time.Now()

	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, `{"error":"missing query param q"}`, http.StatusBadRequest)
		return
	}

	// Step 1: Quick search in memory
	searchStart := time.Now()
	products, total := search(q)
	searchTimeMs := time.Since(searchStart).Milliseconds()

	// Step 2: ✓ FIXED: Use bulkhead to limit concurrent recommendation calls
	recommendations := []interface{}{}
	recTimeMs := int64(0)

	// Try to acquire a slot in the bulkhead
	ctx, cancel := context.WithTimeout(context.Background(), bucketTimeout)
	defer cancel()

	select {
	case recSemaphore <- struct{}{}:  // Acquire slot
		// We have a slot, call the recommendation service
		defer func() { <-recSemaphore }()  // Release slot when done

		recs, recTime, err := recClient.FetchRecommendations(ctx, q)
		recTimeMs = recTime

		if err == nil {
			recommendations = recs
		} else {
			log.Printf("[SEARCH] Recommendation error (but search succeeded): %v", err)
		}

	case <-ctx.Done():
		// Timeout waiting for slot: bulkhead is full
		// ✓ Gracefully degrade: return search results without recommendations
		recTimeMs = bucketTimeout.Milliseconds()
		log.Printf("[SEARCH] Query=%s | Bulkhead FULL, skipping recommendations", q)
	}

	totalTimeMs := time.Since(start).Milliseconds()

	resp := SearchResponse{
		Products:         products,
		TotalFound:       total,
		Recommendations:  recommendations,
		SearchTimeMs:     searchTimeMs,
		RecommendationMs: recTimeMs,
		TotalTimeMs:      totalTimeMs,
		ActiveGoroutines: int(atomic.LoadInt32(&activeGoroutines)),
		Version:          recClient.version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	log.Printf("[SEARCH] Query=%s | Search=%dms | Rec=%dms | Total=%dms | Goroutines=%d",
		q, searchTimeMs, recTimeMs, totalTimeMs, int(atomic.LoadInt32(&activeGoroutines)))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","goroutines":` + strconv.Itoa(int(atomic.LoadInt32(&activeGoroutines))) + `}`))
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active_goroutines": int(atomic.LoadInt32(&activeGoroutines)),
		"timestamp":         time.Now().Unix(),
	})
}

// ── Main ──────────────────────────────────────────────────────────────────────
func main() {
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("HW7: Cascading Failure & Bulkhead Recovery")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	version := "broken"
	if len(os.Args) > 1 {
		version = strings.TrimPrefix(strings.ToLower(os.Args[1]), "--")
	}
	log.Printf("[MAIN] Starting in %s mode\n", version)

	// Start the recommendation service
	StartRecommendationServer()

	// Give recommendation service time to start
	time.Sleep(1 * time.Second)

	// Initialize product store
	log.Println("Generating product catalog...")
	generateProducts(totalProducts)

	// Initialize recommendation client
	recClient = &RecommendationClient{
		baseURL: "http://localhost:8081",
		timeout: 5 * time.Second,
		version: version,
	}

	// Setup HTTP handlers
	if version == "broken" {
		log.Println("[MAIN] Using BROKEN version (no bulkhead protection)")
		http.HandleFunc("/products/search", searchHandlerBroken)
	} else if version == "fixed" {
		log.Println("[MAIN] Using FIXED version (with bulkhead)")
		InitializeBulkhead()
		http.HandleFunc("/products/search", searchHandlerFixed)
	} else {
		log.Fatalf("Unknown version: %s. Use 'broken' or 'fixed'", version)
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/metrics", metricsHandler)

	// Custom transport with pooling to show connection behavior
	http.DefaultClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			MaxConnsPerHost:     100,
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			DisableKeepAlives: false,
		},
	}

	addr := ":8080"
	log.Printf("[MAIN] Search service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
