package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
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

func handle(ctx context.Context, event events.SNSEvent) error {
	_ = ctx

	for _, record := range event.Records {
		var order Order
		if err := json.Unmarshal([]byte(record.SNS.Message), &order); err != nil {
			log.Printf("failed to parse order payload from SNS message: %v", err)
			continue
		}

		start := time.Now()
		log.Printf("lambda processing order %s", order.OrderID)

		// Preserve the same simulated payment delay used in Part II.
		time.Sleep(3 * time.Second)

		log.Printf(
			"lambda completed order %s in %s",
			order.OrderID,
			time.Since(start).Round(time.Millisecond),
		)
	}

	return nil
}

func main() {
	lambda.Start(handle)
}
