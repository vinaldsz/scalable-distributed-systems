package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
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
	Products   []Product `json:"products"`
	TotalFound int       `json:"total_found"`
	SearchTime string    `json:"search_time"`
}

// ── In-memory store ───────────────────────────────────────────────────────────

var store sync.Map // key: int, value: Product

// ── Data generation ───────────────────────────────────────────────────────────

var (
	brands      = []string{"Alpha", "Beta", "Gamma", "Delta", "Echo", "Foxtrot", "Golf", "Hotel"}
	categories  = []string{"Electronics", "Books", "Home", "Sports", "Clothing", "Toys", "Garden", "Automotive"}
	adjectives  = []string{"Pro", "Ultra", "Lite", "Max", "Plus", "Smart", "Advanced", "Essential"}
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
			Description: fmt.Sprintf("%s %s item #%d — durable, reliable, and versatile.", adj, cat, i),
			Brand:       brand,
		}
		store.Store(i, p)
	}
	log.Printf("Generated %d products", n)
}

// ── Search logic ──────────────────────────────────────────────────────────────

const (
	totalProducts = 100_000
	checkLimit    = 100  // fixed computation window
	maxResults    = 20
)

func search(query string) ([]Product, int) {
	q := strings.ToLower(query)
	var results []Product
	totalFound := 0
	checked := 0

	// Iterate product IDs 1..totalProducts; stop after checkLimit checks.
	for id := 1; id <= totalProducts; id++ {
		if checked >= checkLimit {
			break
		}

		val, ok := store.Load(id)
		if !ok {
			continue
		}
		p := val.(Product)
		checked++ // count every product examined, match or not

		if strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(strings.ToLower(p.Category), q) {
			totalFound++
			if len(results) < maxResults {
				results = append(results, p)
			}
		}
	}

	return results, totalFound
}

// ── HTTP handler ──────────────────────────────────────────────────────────────

func searchHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, `{"error":"missing query param q"}`, http.StatusBadRequest)
		return
	}

	products, total := search(q)
	elapsed := time.Since(start)

	resp := SearchResponse{
		Products:   products,
		TotalFound: total,
		SearchTime: elapsed.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	log.Println("Generating product catalog...")
	generateProducts(totalProducts)

	http.HandleFunc("/products/search", searchHandler)
	http.HandleFunc("/health", healthHandler)

	addr := ":8080"
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}