package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Item struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type Order struct {
	OrderID    string    `json:"order_id"`
	CustomerID int       `json:"customer_id"`
	Status     string    `json:"status"`
	Items      []Item    `json:"items"`
	CreatedAt  time.Time `json:"created_at"`
}

var sem chan struct{}

func main() {
	semSize := envInt("PAYMENT_SEM_SIZE", 5)
	sem = make(chan struct{}, semSize)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/orders/sync", syncHandler)

	log.Printf("Order receiver starting on :%s (payment_sem_size=%d)", port, semSize)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ok"}`)
}

func syncHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	order.OrderID = defaultID(order.OrderID)
	order.CreatedAt = time.Now()
	order.Status = "processing"

	// Buffered semaphore limits concurrent payment processing.
	sem <- struct{}{}
	defer func() { <-sem }()

	// Simulate payment verification bottleneck.
	time.Sleep(3 * time.Second)

	order.Status = "completed"
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(order)
}

func defaultID(id string) string {
	if id != "" {
		return id
	}
	return fmt.Sprintf("ord-%d", time.Now().UnixNano())
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
