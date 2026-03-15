package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

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

// SNS wraps the actual message in a Message field
type SNSEnvelope struct {
	Message string `json:"Message"`
}

var (
	sqsClient *sqs.SQS
	queueURL  string
)

func processOrder(order Order) {
	log.Printf("Processing order %s for customer %d", order.OrderID, order.CustomerID)
	time.Sleep(3 * time.Second) // Simulated payment verification
	log.Printf("Completed order %s", order.OrderID)
}

func worker(id int, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("Worker %d started", id)

	for {
		result, err := sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: aws.Int64(10),
			WaitTimeSeconds:     aws.Int64(20), // Long polling
			VisibilityTimeout:   aws.Int64(30),
		})
		if err != nil {
			log.Printf("Worker %d receive error: %v", id, err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, msg := range result.Messages {
			// SNS wraps the order JSON in an envelope
			var envelope SNSEnvelope
			if err := json.Unmarshal([]byte(*msg.Body), &envelope); err != nil {
				log.Printf("Worker %d envelope parse error: %v", id, err)
				deleteMessage(msg.ReceiptHandle)
				continue
			}

			var order Order
			if err := json.Unmarshal([]byte(envelope.Message), &order); err != nil {
				log.Printf("Worker %d order parse error: %v", id, err)
				deleteMessage(msg.ReceiptHandle)
				continue
			}

			processOrder(order)
			deleteMessage(msg.ReceiptHandle)
		}
	}
}

func deleteMessage(receiptHandle *string) {
	_, err := sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: receiptHandle,
	})
	if err != nil {
		log.Printf("Delete error: %v", err)
	}
}

func main() {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	queueURL = os.Getenv("SQS_QUEUE_URL")
	if queueURL == "" {
		log.Fatal("SQS_QUEUE_URL environment variable required")
	}

	numWorkers := 1
	if w := os.Getenv("NUM_WORKERS"); w != "" {
		if n, err := strconv.Atoi(w); err == nil {
			numWorkers = n
		}
	}

	sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		log.Fatalf("AWS session error: %v", err)
	}
	sqsClient = sqs.New(sess)

	log.Printf("Starting %d worker goroutines, polling %s", numWorkers, queueURL)

	var wg sync.WaitGroup
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, &wg)
	}
	wg.Wait() // Runs forever
}
