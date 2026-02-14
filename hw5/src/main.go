package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Product represents a product as defined in the OpenAPI spec
type Product struct {
	ProductID    int    `json:"product_id"`
	SKU          string `json:"sku"`
	Manufacturer string `json:"manufacturer"`
	CategoryID   int    `json:"category_id"`
	Weight       int    `json:"weight"`
	SomeOtherID  int    `json:"some_other_id"`
}

// ErrorResponse represents an error response as defined in the OpenAPI spec
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ProductStore manages our in-memory product storage
type ProductStore struct {
	products map[int]Product
	mu       sync.RWMutex
}

// NewProductStore creates a new product store
func NewProductStore() *ProductStore {
	return &ProductStore{
		products: make(map[int]Product),
	}
}

// Global store instance
var store = NewProductStore()

func main() {
	// Register route handlers
	http.HandleFunc("/products/", handleProducts)

	port := ":8080"
	fmt.Printf("Product API Server starting on port %s\n", port)
	fmt.Printf("Endpoints:\n")
	fmt.Printf("  GET    /products/{productId}\n")
	fmt.Printf("  POST   /products/{productId}/details\n")
	log.Fatal(http.ListenAndServe(port, nil))
}

// handleProducts routes requests to appropriate handlers
func handleProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse the URL path
	path := strings.TrimPrefix(r.URL.Path, "/products/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Endpoint not found", "")
		return
	}

	// Extract product ID
	productID, err := strconv.Atoi(parts[0])
	if err != nil || productID < 1 {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", 
			"Invalid product ID", "Product ID must be a positive integer")
		return
	}

	// Route based on path and method
	if len(parts) == 1 {
		// /products/{productId}
		if r.Method == http.MethodGet {
			getProduct(w, r, productID)
		} else {
			sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", 
				"Method not allowed", fmt.Sprintf("%s method is not supported for this endpoint", r.Method))
		}
	} else if len(parts) == 2 && parts[1] == "details" {
		// /products/{productId}/details
		if r.Method == http.MethodPost {
			addProductDetails(w, r, productID)
		} else {
			sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", 
				"Method not allowed", fmt.Sprintf("%s method is not supported for this endpoint", r.Method))
		}
	} else {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "Endpoint not found", "")
	}
}

// getProduct handles GET /products/{productId}
func getProduct(w http.ResponseWriter, r *http.Request, productID int) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	product, exists := store.products[productID]
	if !exists {
		sendError(w, http.StatusNotFound, "NOT_FOUND", 
			"Product not found", fmt.Sprintf("Product with ID %d does not exist", productID))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(product)
}

// addProductDetails handles POST /products/{productId}/details
func addProductDetails(w http.ResponseWriter, r *http.Request, productID int) {
	var product Product

	// Decode JSON body
	err := json.NewDecoder(r.Body).Decode(&product)
	if err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", 
			"Invalid JSON format", "Request body must be valid JSON")
		return
	}

	// Validate that product_id in body matches URL parameter
	if product.ProductID != 0 && product.ProductID != productID {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", 
			"Product ID mismatch", 
			fmt.Sprintf("Product ID in URL (%d) does not match product ID in body (%d)", productID, product.ProductID))
		return
	}

	// Set the product ID from URL if not provided in body
	product.ProductID = productID

	// Validate the product
	if err := validateProduct(product); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_INPUT", 
			"Invalid input data", err.Error())
		return
	}

	// Store the product
	store.mu.Lock()
	store.products[productID] = product
	store.mu.Unlock()

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// validateProduct validates product fields according to OpenAPI spec
func validateProduct(p Product) error {
	// product_id: minimum 1
	if p.ProductID < 1 {
		return fmt.Errorf("product_id must be a positive integer (minimum 1)")
	}

	// sku: minLength 1, maxLength 100
	if len(strings.TrimSpace(p.SKU)) == 0 {
		return fmt.Errorf("sku is required and cannot be empty")
	}
	if len(p.SKU) > 100 {
		return fmt.Errorf("sku must not exceed 100 characters")
	}

	// manufacturer: minLength 1, maxLength 200
	if len(strings.TrimSpace(p.Manufacturer)) == 0 {
		return fmt.Errorf("manufacturer is required and cannot be empty")
	}
	if len(p.Manufacturer) > 200 {
		return fmt.Errorf("manufacturer must not exceed 200 characters")
	}

	// category_id: minimum 1
	if p.CategoryID < 1 {
		return fmt.Errorf("category_id must be a positive integer (minimum 1)")
	}

	// weight: minimum 0
	if p.Weight < 0 {
		return fmt.Errorf("weight must be greater than or equal to 0")
	}

	// some_other_id: minimum 1
	if p.SomeOtherID < 1 {
		return fmt.Errorf("some_other_id must be a positive integer (minimum 1)")
	}

	return nil
}

// sendError sends an error response in the format specified by the OpenAPI spec
func sendError(w http.ResponseWriter, statusCode int, errorCode, message, details string) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errorCode,
		Message: message,
		Details: details,
	})
}