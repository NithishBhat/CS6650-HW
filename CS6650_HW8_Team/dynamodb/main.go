package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var dbClient *dynamodb.Client
var tableName string

func main() {
	var err error
	dbClient, tableName, err = initDynamoDB()
	if err != nil {
		log.Fatalf("Failed to initialize DynamoDB: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/shopping-carts", handleShoppingCarts)
	mux.HandleFunc("/shopping-carts/", handleShoppingCartsWithID)
	mux.HandleFunc("/health", handleHealth)

	log.Println("DynamoDB server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func initDynamoDB() (*dynamodb.Client, string, error) {
	region := getEnv("AWS_REGION", "us-west-2")
	table := getEnv("DYNAMODB_TABLE", "shopping-carts-dynamodb")

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, "", err
	}

	client := dynamodb.NewFromConfig(cfg)
	return client, table, nil
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
	respondJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
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

	cartID := parts[0]

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

type AddItemRequest struct {
	ProductID uint64 `json:"product_id"`
	Quantity  uint32 `json:"quantity"`
}

type ItemResponse struct {
	ItemID    string `json:"item_id"`
	ProductID uint64 `json:"product_id"`
	Quantity  uint32 `json:"quantity"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CartResponse struct {
	CartID     string         `json:"cart_id"`
	CustomerID uint64         `json:"customer_id"`
	Status     string         `json:"status"`
	Items      []ItemResponse `json:"items"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

// DynamoDB row types

type CartMetaItem struct {
	PK         string `dynamodbav:"pk"`
	SK         string `dynamodbav:"sk"`
	CartID     string `dynamodbav:"cart_id"`
	CustomerID uint64 `dynamodbav:"customer_id"`
	Status     string `dynamodbav:"status"`
	CreatedAt  string `dynamodbav:"created_at"`
	UpdatedAt  string `dynamodbav:"updated_at"`
}

type CartItemRow struct {
	PK         string `dynamodbav:"pk"`
	SK         string `dynamodbav:"sk"`
	ItemID     string `dynamodbav:"item_id"`
	ProductID  uint64 `dynamodbav:"product_id"`
	Quantity   uint32 `dynamodbav:"quantity"`
	CreatedAt  string `dynamodbav:"created_at"`
	UpdatedAt  string `dynamodbav:"updated_at"`
}

// --- Helpers for key format ---

func cartPK(cartID string) string {
	return "CART#" + cartID
}

func metaSK() string {
	return "META"
}

func itemSK(productID uint64) string {
	return "ITEM#" + strconv.FormatUint(productID, 10)
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func generateCartID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
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

	cartID := generateCartID()
	ts := nowISO()

	meta := CartMetaItem{
		PK:         cartPK(cartID),
		SK:         metaSK(),
		CartID:     cartID,
		CustomerID: req.CustomerID,
		Status:     "active",
		CreatedAt:  ts,
		UpdatedAt:  ts,
	}

	item, err := attributevalue.MarshalMap(meta)
	if err != nil {
		log.Printf("Error marshaling cart meta: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create cart")
		return
	}

	_, err = dbClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName:           aws.String(tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(pk) AND attribute_not_exists(sk)"),
	})
	if err != nil {
		log.Printf("Error creating cart in DynamoDB: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create cart")
		return
	}

	resp := CartResponse{
		CartID:     cartID,
		CustomerID: req.CustomerID,
		Status:     "active",
		Items:      []ItemResponse{},
		CreatedAt:  ts,
		UpdatedAt:  ts,
	}

	respondJSON(w, http.StatusCreated, resp)
}

// GET /shopping-carts/{id}
func getCart(w http.ResponseWriter, r *http.Request, cartID string) {
	consistent := r.URL.Query().Get("consistent") == "true"

	out, err := dbClient.Query(context.Background(), &dynamodb.QueryInput{
		TableName: aws.String(tableName),
		KeyConditionExpression: aws.String("pk = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: cartPK(cartID)},
		},
		ConsistentRead: aws.Bool(consistent),
	})
	if err != nil {
		log.Printf("Error querying cart from DynamoDB: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to retrieve cart")
		return
	}

	if len(out.Items) == 0 {
		respondError(w, http.StatusNotFound, "Cart not found")
		return
	}

	var meta *CartMetaItem
	items := []ItemResponse{}

	for _, row := range out.Items {
		var sk string
		if v, ok := row["sk"].(*types.AttributeValueMemberS); ok {
			sk = v.Value
		}

		if sk == "META" {
			var m CartMetaItem
			if err := attributevalue.UnmarshalMap(row, &m); err != nil {
				log.Printf("Error unmarshaling cart meta: %v", err)
				respondError(w, http.StatusInternalServerError, "Failed to read cart")
				return
			}
			meta = &m
			continue
		}

		if strings.HasPrefix(sk, "ITEM#") {
			var itemRow CartItemRow
			if err := attributevalue.UnmarshalMap(row, &itemRow); err != nil {
				log.Printf("Error unmarshaling cart item: %v", err)
				respondError(w, http.StatusInternalServerError, "Failed to read cart items")
				return
			}

			items = append(items, ItemResponse{
				ItemID:    itemRow.ItemID,
				ProductID: itemRow.ProductID,
				Quantity:  itemRow.Quantity,
				CreatedAt: itemRow.CreatedAt,
				UpdatedAt: itemRow.UpdatedAt,
			})
		}
	}

	if meta == nil {
		respondError(w, http.StatusNotFound, "Cart not found")
		return
	}

	resp := CartResponse{
		CartID:     meta.CartID,
		CustomerID: meta.CustomerID,
		Status:     meta.Status,
		Items:      items,
		CreatedAt:  meta.CreatedAt,
		UpdatedAt:  meta.UpdatedAt,
	}

	respondJSON(w, http.StatusOK, resp)
}

// POST /shopping-carts/{id}/items
func addCartItem(w http.ResponseWriter, r *http.Request, cartID string) {
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

	// First verify cart exists by reading META row
	metaOut, err := dbClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: cartPK(cartID)},
			"sk": &types.AttributeValueMemberS{Value: metaSK()},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		log.Printf("Error reading cart meta: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to process request")
		return
	}
	if len(metaOut.Item) == 0 {
		respondError(w, http.StatusNotFound, "Cart not found")
		return
	}

	var meta CartMetaItem
	if err := attributevalue.UnmarshalMap(metaOut.Item, &meta); err != nil {
		log.Printf("Error unmarshaling cart meta: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to process request")
		return
	}

	if meta.Status != "active" {
		respondError(w, http.StatusConflict, "Cart is already checked out")
		return
	}

	now := nowISO()
	itemID := "item_" + strconv.FormatUint(req.ProductID, 10)

	// Check whether item already exists so we can preserve created_at if present
	existingOut, err := dbClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: cartPK(cartID)},
			"sk": &types.AttributeValueMemberS{Value: itemSK(req.ProductID)},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		log.Printf("Error checking existing cart item: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to add item")
		return
	}

	createdAt := now
	if len(existingOut.Item) > 0 {
		var existing CartItemRow
		if err := attributevalue.UnmarshalMap(existingOut.Item, &existing); err == nil {
			createdAt = existing.CreatedAt
			itemID = existing.ItemID
		}
	}

	itemRow := CartItemRow{
		PK:        cartPK(cartID),
		SK:        itemSK(req.ProductID),
		ItemID:    itemID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		CreatedAt: createdAt,
		UpdatedAt: now,
	}

	itemMap, err := attributevalue.MarshalMap(itemRow)
	if err != nil {
		log.Printf("Error marshaling cart item: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to add item")
		return
	}

	_, err = dbClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      itemMap,
	})
	if err != nil {
		log.Printf("Error writing cart item: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to add item")
		return
	}

	// Touch META.updated_at
	_, err = dbClient.UpdateItem(context.Background(), &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: cartPK(cartID)},
			"sk": &types.AttributeValueMemberS{Value: metaSK()},
		},
		UpdateExpression: aws.String("SET updated_at = :u"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":u": &types.AttributeValueMemberS{Value: now},
		},
	})
	if err != nil {
		log.Printf("Error updating cart meta timestamp: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to add item")
		return
	}

	resp := ItemResponse{
		ItemID:    itemRow.ItemID,
		ProductID: itemRow.ProductID,
		Quantity:  itemRow.Quantity,
		CreatedAt: itemRow.CreatedAt,
		UpdatedAt: itemRow.UpdatedAt,
	}

	respondJSON(w, http.StatusCreated, resp)
}

// --- Response helpers ---

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}