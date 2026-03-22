package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func main() {
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/shopping-carts", handleShoppingCarts)
	mux.HandleFunc("/shopping-carts/", handleShoppingCartsWithID)
	mux.HandleFunc("/health", handleHealth)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

// initDB connects to MySQL and auto-creates tables.
func initDB() (*sql.DB, error) {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("DB_NAME"),
    )

    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }

    // Connection pool settings
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)

    // Auto-create tables (safe to run every time)
    if err := runSchema(db); err != nil {
        return nil, err
    }

    return db, nil
}

func runSchema(db *sql.DB) error {
    cartsTable := `
    CREATE TABLE IF NOT EXISTS shopping_carts (
        cart_id     BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
        customer_id BIGINT UNSIGNED NOT NULL,
        status      ENUM('active', 'checked_out') NOT NULL DEFAULT 'active',
        created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
        updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
        PRIMARY KEY (cart_id),
        KEY idx_carts_customer_created (customer_id, created_at DESC)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;`

    itemsTable := `
    CREATE TABLE IF NOT EXISTS cart_items (
        item_id    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
        cart_id    BIGINT UNSIGNED NOT NULL,
        product_id BIGINT UNSIGNED NOT NULL,
        quantity   INT UNSIGNED    NOT NULL DEFAULT 1,
        created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
        updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
        PRIMARY KEY (item_id),
        UNIQUE KEY uk_cart_product (cart_id, product_id),
        CONSTRAINT fk_cart_items_cart FOREIGN KEY (cart_id)
            REFERENCES shopping_carts (cart_id) ON DELETE CASCADE
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;`

    if _, err := db.Exec(cartsTable); err != nil {
        return err
    }
    _, err := db.Exec(itemsTable)
    return err
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// --- Routing ---

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func handleShoppingCarts(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		createCart(w, r)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleShoppingCartsWithID(w http.ResponseWriter, r *http.Request) {
	// Parse path: /shopping-carts/{id} or /shopping-carts/{id}/items
	path := strings.TrimPrefix(r.URL.Path, "/shopping-carts/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Cart ID required", http.StatusBadRequest)
		return
	}

	cartID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid cart ID", http.StatusBadRequest)
		return
	}

	// /shopping-carts/{id}/items
	if len(parts) == 2 && parts[1] == "items" {
		if r.Method == http.MethodPost {
			addCartItem(w, r, cartID)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// /shopping-carts/{id}
	if len(parts) == 1 {
		if r.Method == http.MethodGet {
			getCart(w, r, cartID)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.NotFound(w, r)
}

// --- Request/Response types ---

type CreateCartRequest struct {
	CustomerID uint64 `json:"customer_id"`
}

type CartResponse struct {
	CartID     uint64         `json:"cart_id"`
	CustomerID uint64         `json:"customer_id"`
	Status     string         `json:"status"`
	Items      []ItemResponse `json:"items"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type ItemResponse struct {
	ItemID    uint64    `json:"item_id"`
	ProductID uint64    `json:"product_id"`
	Quantity  uint32    `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AddItemRequest struct {
	ProductID uint64 `json:"product_id"`
	Quantity  uint32 `json:"quantity"`
}

// --- Handlers ---

// POST /shopping-carts
func createCart(w http.ResponseWriter, r *http.Request) {
	var req CreateCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.CustomerID == 0 {
		respondError(w, http.StatusBadRequest, "customer_id is required and must be > 0")
		return
	}

	result, err := db.Exec("INSERT INTO shopping_carts (customer_id) VALUES (?)", req.CustomerID)
	if err != nil {
		log.Printf("Error creating cart: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create cart")
		return
	}

	id, _ := result.LastInsertId()

	// Fetch the created cart
	var cart CartResponse
	err = db.QueryRow(
		"SELECT cart_id, customer_id, status, created_at, updated_at FROM shopping_carts WHERE cart_id = ?", id,
	).Scan(&cart.CartID, &cart.CustomerID, &cart.Status, &cart.CreatedAt, &cart.UpdatedAt)
	if err != nil {
		log.Printf("Error fetching created cart: %v", err)
		respondError(w, http.StatusInternalServerError, "Cart created but failed to retrieve")
		return
	}
	cart.Items = []ItemResponse{}

	respondJSON(w, http.StatusCreated, cart)
}

// GET /shopping-carts/{id}
func getCart(w http.ResponseWriter, r *http.Request, cartID uint64) {
	var cart CartResponse
	err := db.QueryRow(
		"SELECT cart_id, customer_id, status, created_at, updated_at FROM shopping_carts WHERE cart_id = ?", cartID,
	).Scan(&cart.CartID, &cart.CustomerID, &cart.Status, &cart.CreatedAt, &cart.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "Cart not found")
			return
		}
		log.Printf("Error querying cart: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to retrieve cart")
		return
	}

	// Fetch items with a JOIN-efficient query
	rows, err := db.Query(
		"SELECT item_id, product_id, quantity, created_at, updated_at FROM cart_items WHERE cart_id = ? ORDER BY item_id", cartID,
	)
	if err != nil {
		log.Printf("Error querying cart items: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to retrieve cart items")
		return
	}
	defer rows.Close()

	cart.Items = []ItemResponse{}
	for rows.Next() {
		var item ItemResponse
		if err := rows.Scan(&item.ItemID, &item.ProductID, &item.Quantity, &item.CreatedAt, &item.UpdatedAt); err != nil {
			log.Printf("Error scanning cart item: %v", err)
			respondError(w, http.StatusInternalServerError, "Failed to read cart items")
			return
		}
		cart.Items = append(cart.Items, item)
	}

	respondJSON(w, http.StatusOK, cart)
}

// POST /shopping-carts/{id}/items
func addCartItem(w http.ResponseWriter, r *http.Request, cartID uint64) {
	var req AddItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.ProductID == 0 {
		respondError(w, http.StatusBadRequest, "product_id is required and must be > 0")
		return
	}
	if req.Quantity == 0 {
		req.Quantity = 1
	}

	// Use a transaction to verify cart exists and upsert the item atomically
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to process request")
		return
	}
	defer tx.Rollback()

	// Verify cart exists and is active
	var status string
	err = tx.QueryRow("SELECT status FROM shopping_carts WHERE cart_id = ?", cartID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "Cart not found")
			return
		}
		log.Printf("Error checking cart: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to process request")
		return
	}
	if status != "active" {
		respondError(w, http.StatusConflict, "Cart is already checked out")
		return
	}

	// Upsert: insert or update quantity if product already in cart
	_, err = tx.Exec(
		`INSERT INTO cart_items (cart_id, product_id, quantity)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE quantity = VALUES(quantity)`,
		cartID, req.ProductID, req.Quantity,
	)
	if err != nil {
		log.Printf("Error upserting cart item: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to add item")
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to save item")
		return
	}

	// Return the updated item
	var item ItemResponse
	err = db.QueryRow(
		"SELECT item_id, product_id, quantity, created_at, updated_at FROM cart_items WHERE cart_id = ? AND product_id = ?",
		cartID, req.ProductID,
	).Scan(&item.ItemID, &item.ProductID, &item.Quantity, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		log.Printf("Error fetching upserted item: %v", err)
		respondError(w, http.StatusInternalServerError, "Item saved but failed to retrieve")
		return
	}

	respondJSON(w, http.StatusCreated, item)
}

// --- Helpers ---

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
