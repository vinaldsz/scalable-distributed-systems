package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SNSEnvelope struct {
	Message string `json:"Message"`
}

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

var (
	sqsClient *sqs.Client
	queueURL  string
)

func main() {
	queueURL = os.Getenv("SQS_QUEUE_URL")
	if queueURL == "" {
		log.Fatal("SQS_QUEUE_URL environment variable is required")
	}

	numWorkers := envInt("NUM_WORKERS", 1)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}
	sqsClient = sqs.NewFromConfig(cfg)

	jobs := make(chan sqstypes.Message, numWorkers*2)
	for i := 0; i < numWorkers; i++ {
		go worker(i, jobs)
	}
	go poller(jobs)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	log.Printf("order processor starting on :%s (workers=%d)", port, numWorkers)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func poller(jobs chan<- sqstypes.Message) {
	for {
		result, err := sqsClient.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     20,
		})
		if err != nil {
			log.Printf("ReceiveMessage error: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for _, msg := range result.Messages {
			jobs <- msg
		}
	}
}

func worker(id int, jobs <-chan sqstypes.Message) {
	for msg := range jobs {
		processMessage(id, msg)
	}
}

func processMessage(workerID int, msg sqstypes.Message) {
	var envelope SNSEnvelope
	if err := json.Unmarshal([]byte(aws.ToString(msg.Body)), &envelope); err != nil {
		log.Printf("worker %d envelope parse error: %v", workerID, err)
		deleteMessage(msg)
		return
	}

	var order Order
	if err := json.Unmarshal([]byte(envelope.Message), &order); err != nil {
		log.Printf("worker %d order parse error: %v", workerID, err)
		deleteMessage(msg)
		return
	}

	log.Printf("worker %d processing order %s", workerID, order.OrderID)
	time.Sleep(3 * time.Second)
	log.Printf("worker %d completed order %s", workerID, order.OrderID)
	deleteMessage(msg)
}

func deleteMessage(msg sqstypes.Message) {
	_, err := sqsClient.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		log.Printf("DeleteMessage error: %v", err)
	}
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
