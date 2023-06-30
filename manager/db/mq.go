package db

import (
	"encoding/json"
	"github.com/SparkSecurity/wakizashi/manager/config"
	"github.com/SparkSecurity/wakizashi/manager/util"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"time"
)

var MQConn *amqp.Connection
var MQChan *amqp.Channel

func MQConnect() {
	// Connect mq
	var err error
	MQConn, err = amqp.Dial(config.Config.MQURI)
	if err != nil {
		panic(err)
	}

	MQChan, err = MQConn.Channel()
	if err != nil {
		panic(err)
	}
	err = MQChan.ExchangeDeclare("page.events", "direct", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	// queue for scraping page tasks
	_, err = MQChan.QueueDeclare(
		"page.scrape",
		true,
		false,
		false,
		false,
		map[string]interface{}{
			"x-dead-letter-exchange":    "page.events",
			"x-dead-letter-routing-key": "response",
		},
	)
	if err != nil {
		panic(err)
	}

	// queue for scrape responses
	_, err = MQChan.QueueDeclare("page.response", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	// Bind queues to exchange
	err = MQChan.QueueBind("page.scrape", "scrape", "page.events", false, nil)
	if err != nil {
		panic(err)
	}
	err = MQChan.QueueBind("page.response", "response", "page.events", false, nil)
	if err != nil {
		panic(err)
	}

	// exchange for retry tasks
	// x-retry-count = 0 -> page.response
	// otherwise -> page.events key=scrape
	err = MQChan.ExchangeDeclare("page.events.retry",
		"headers",
		false,
		false,
		false,
		false,
		map[string]interface{}{
			"alternate-exchange": "page.events",
		},
	)
	if err != nil {
		panic(err)
	}

	err = MQChan.QueueBind("page.response",
		"scrape",
		"page.events.retry",
		false,
		map[string]interface{}{
			"retry-count": 0,
			"x-match":     "all",
		},
	)
	if err != nil {
		panic(err)
	}
}

func MQDisconnect() {
	_ = MQChan.Close()
	_ = MQConn.Close()
}

// MQConsumeResponse fetches the responses pushed by the workers
// Accepts a handler function that will be called when a response is received
// handler should process the response and update the page status in the database
func MQConsumeResponse(handler func(success bool, task ScrapeTask)) {
	// Fetch all the responses pushed by the workers
	msgs, err := MQChan.Consume(
		"page.response",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}

	for d := range msgs {
		d := d

		// Try to parse the message. Set the success flag to 1 if the message is parsed successfullyOnly
		// Remove the message from the queue if it is parsed successfully or cannot be parsed
		go func() {
			var task ScrapeTask
			err := json.Unmarshal(d.Body, &task)
			if err != nil {
				log.Println(err.Error(), string(d.Body))
				_ = d.Ack(false) // remove the message from the queue
				return
			}
			handler(d.Headers["x-success"].(int32) == 1, task) // set the success flag to 1 and call the handler
			_ = d.Ack(false)                                   // remove the message from the queue
		}()
	}
}

type ScrapeTask struct {
	ID       string   `json:"id"`
	Url      string   `json:"url"`
	Response string   `json:"response,omitempty"`
	Error    []string `json:"error,omitempty"`
	Browser  bool     `json:"browser"`
}

// PublishScrapeTask publishes a scrape task to the queue, allowing the worker to scrape the page
func PublishScrapeTask(task ScrapeTask) error {
	// Convert the task to json
	jsonBytes, err := json.Marshal(task)
	if err != nil {
		return err
	}

	// Publish the task to the queue, with a timeout of 5 seconds
	ctx, cancel := util.TimeoutContext(5 * time.Second)
	defer cancel()
	err = MQChan.PublishWithContext(ctx, "page.events", "scrape", false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Headers: map[string]interface{}{
			"retry-count": 3,
			"x-success":   0,
		},
		Body: jsonBytes,
	})
	if err != nil {
		return err
	}

	return nil
}
