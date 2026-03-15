package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/google/uuid"
)

// ---------- Models ----------

type Item struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type Order struct {
	OrderID    string    `json:"order_id"`
	CustomerID int       `json:"customer_id"`
	Status     string    `json:"status"`
	Items      []Item    `json:"items"`
	CreatedAt  time.Time `json:"created_at"`
}

type Product struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ---------- Globals ----------

var (
	productsMap sync.Map

	// SIMULATED BOTTLENECK: buffered channel as semaphore.
	// Capacity 15 allows ~5 orders/sec when each takes 3s (5 * 3 = 15 slots).
	paymentSema = make(chan struct{}, 15)

	snsClient *sns.SNS
	topicARN  string
)

// ---------- Data ----------

func generateData() {
	start := time.Now()
	for i := 1; i <= 100000; i++ {
		productsMap.Store(i, Product{
			ID:   i,
			Name: fmt.Sprintf("Product-%d", i),
		})
	}
	log.Printf("Generated 100,000 products in %v", time.Since(start))
}

// ---------- Payment simulation ----------

func simulatePayment() {
	// Acquire semaphore slot — blocks if all 15 are in use
	paymentSema <- struct{}{}
	defer func() { <-paymentSema }()

	time.Sleep(3 * time.Second)
}

// ---------- Handlers ----------

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func syncOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CustomerID int    `json:"customer_id"`
		Items      []Item `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	order := Order{
		OrderID:    uuid.New().String(),
		CustomerID: req.CustomerID,
		Status:     "processing",
		Items:      req.Items,
		CreatedAt:  time.Now(),
	}

	// Synchronous payment — blocks the response
	simulatePayment()

	order.Status = "completed"
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(order)
}

func asyncOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CustomerID int    `json:"customer_id"`
		Items      []Item `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	order := Order{
		OrderID:    uuid.New().String(),
		CustomerID: req.CustomerID,
		Status:     "pending",
		Items:      req.Items,
		CreatedAt:  time.Now(),
	}

	// Publish to SNS — non-blocking
	orderJSON, _ := json.Marshal(order)
	_, err := snsClient.Publish(&sns.PublishInput{
		TopicArn: aws.String(topicARN),
		Message:  aws.String(string(orderJSON)),
	})
	if err != nil {
		log.Printf("SNS publish error: %v", err)
		http.Error(w, "Failed to queue order", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(order)
}

// ---------- Main ----------

func main() {
	generateData()

	// AWS setup
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	topicARN = os.Getenv("SNS_TOPIC_ARN")

	sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		log.Fatalf("AWS session error: %v", err)
	}
	snsClient = sns.New(sess)

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/orders/sync", syncOrderHandler)
	http.HandleFunc("/orders/async", asyncOrderHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("API listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
