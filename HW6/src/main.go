package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Product structure as required by Part 2
type Product struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Brand       string `json:"brand"`
}

var (
	// Using sync.Map for thread-safe storage as required
	inventory sync.Map
)

func init() {
	// Data Generation: Generate 100,000 products at startup
	categories := []string{"Electronics", "Books", "Home", "Clothing", "Garden"}
	brands := []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"}

	for i := 1; i <= 100000; i++ {
		brand := brands[i%len(brands)]
		category := categories[i%len(categories)]

		p := Product{
			ID:          i,
			Name:        "Product " + brand + " " + strconv.Itoa(i), // Format: "Product [Brand] [ID]"
			Category:    category,                                   // Rotate through categories using modulo
			Description: "A high-quality product from our " + category + " line.",
			Brand:       brand,
		}
		inventory.Store(i, p)
	}
}

// healthHandler provides a simple 200 OK for the ALB health checks
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// searchHandler implements the bounded iteration logic
func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(r.URL.Query().Get("q"))

	var results []Product
	totalFound := 0
	checkedCount := 0

	// sync.Map.Range iterates through the map
	inventory.Range(func(key, value any) bool {
		// Rule: Increment counter for EVERY product checked
		checkedCount++

		p := value.(Product)
		nameMatch := strings.Contains(strings.ToLower(p.Name), query)
		categoryMatch := strings.Contains(strings.ToLower(p.Category), query)

		if nameMatch || categoryMatch {
			totalFound++
			// Rule: Return max 20 results
			if len(results) < 20 {
				results = append(results, p)
			}
		}

		// Rule: Check exactly 100 products then stop
		return checkedCount < 100
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"products":    results,
		"total_found": totalFound,
		"search_time": "0s", // Optional placeholder
	})
}

// productHandler handles individual product retrieval and updates
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

	// HANDLE WRITES (POST)
	if r.Method == http.MethodPost {
		var p Product
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid input data", http.StatusBadRequest)
			return
		}

		if p.ID != idParam {
			http.Error(w, "Product ID mismatch", http.StatusBadRequest)
			return
		}

		inventory.Store(idParam, p)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// HANDLE READS (GET)
	if r.Method == http.MethodGet {
		val, exists := inventory.Load(idParam)

		if !exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "NOT_FOUND", "message": "Product not found"})
			return
		}

		product := val.(Product)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(product)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func main() {
	// Health check endpoint for ALB
	http.HandleFunc("/health", healthHandler)

	// Search endpoint: /products/search?q={query}
	http.HandleFunc("/products/search", searchHandler)

	// Individual product endpoints
	http.HandleFunc("/products/", productHandler)

	http.ListenAndServe(":8080", nil)
}