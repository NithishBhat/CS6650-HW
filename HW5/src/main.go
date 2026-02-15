package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Product defines the structure from api.yaml
type Product struct {
	ProductID    int    `json:"product_id"`
	SKU          string `json:"sku"`
	Manufacturer string `json:"manufacturer"`
	CategoryID   int    `json:"category_id"`
	Weight       int    `json:"weight"`
	OtherID      int    `json:"some_other_id"`
}

var (
	inventory = make(map[int]Product)
	// RWMutex is the "Wise Choice" for read-heavy APIs
	mu sync.RWMutex
)

func productHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	
	idParam, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	// HANDLE WRITES (POST) - Needs Exclusive Lock
	if r.Method == http.MethodPost {
		var p Product
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid input data", http.StatusBadRequest)
			return
		}

		if p.ProductID != idParam {
			http.Error(w, "Product ID mismatch", http.StatusBadRequest)
			return
		}

		mu.Lock() // Stops EVERYONE (Reads and Writes)
		inventory[idParam] = p
		mu.Unlock()

		w.WriteHeader(http.StatusNoContent)
		return
	}
	
	// HANDLE READS (GET) - Needs Read Lock
	if r.Method == http.MethodGet {
		mu.RLock() // Allows other Readers to enter, but stops Writers
		product, exists := inventory[idParam]
		mu.RUnlock()

		if !exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "NOT_FOUND", "message": "Product not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(product)
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func main() {
	http.HandleFunc("/products/", productHandler)
	http.ListenAndServe(":8080", nil)
}