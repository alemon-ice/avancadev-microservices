package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/wesleywillians/go-rabbitmq/queue"

	"github.com/joho/godotenv"
	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"
)

type Result struct {
	Status string
}

type Order struct {
	ID       uuid.UUID
	Coupon   string
	CcNumber string
}

func NewOrder() Order {
	return Order{ID: uuid.NewV4()}
}

const (
	InvalidCoupon   = "invalid"
	UsedCoupon      = "used"
	ValidCoupon     = "valid"
	ConnectionError = "connection error"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env")
	}
}

func main() {
	messageChannel := make(chan amqp.Delivery)

	rabbitMQ := queue.NewRabbitMQ()
	ch := rabbitMQ.Connect()
	defer ch.Close()

	rabbitMQ.Consume(messageChannel)

	for msg := range messageChannel {
		process(msg)
	}
}

func process(msg amqp.Delivery) {
	order := NewOrder()
	json.Unmarshal(msg.Body, &order)

	resultCoupon := makeHTTPCall("http://localhost:9092", order.Coupon)

	switch resultCoupon.Status {
	case InvalidCoupon:
		log.Println("Order - ", order.ID, ": Invalid coupon!")

	case UsedCoupon:
		log.Println("Order - ", order.ID, ": Coupon has already been used!")

	case ConnectionError:
		msg.Reject(false)
		log.Println("Order - ", order.ID, ": Could not process!")

	case ValidCoupon:
		log.Println("Order - ", order.ID, ": Processed!")
	}
}

func makeHTTPCall(urlMicroservice string, coupon string) Result {
	values := url.Values{}
	values.Add("coupon", coupon)

	// retryClient := retryablehttp.NewClient()
	// retryClient.RetryMax = 3

	// res, err := retryClient.PostForm(urlMicroservice, values)
	res, err := http.PostForm(urlMicroservice, values)
	if err != nil {
		result := Result{Status: ConnectionError}
		return result
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Error processing result")
	}

	result := Result{}

	json.Unmarshal(data, &result)

	return result
}
