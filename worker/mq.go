package main

import (
	"context"
	"encoding/json"
	"github.com/SparkSecurity/wakizashi/worker/config"
	"github.com/SparkSecurity/wakizashi/worker/scrape"
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
}

func MQDisconnect() {
	_ = MQChan.Close()
	_ = MQConn.Close()
}

func MQConsume(handler func(task *scrape.Task) error) {
	err := MQChan.Qos(config.Config.PrefetchCount, 0, false)
	if err != nil {
		panic(err)
	}
	msgs, err := MQChan.Consume("page.scrape", "", false, false, false, false, nil)
	if err != nil {
		panic(err)
	}
	for d := range msgs {
		d := d
		go func() {
			var task scrape.Task
			err := json.Unmarshal(d.Body, &task)
			if err != nil {
				retry(&d)
				return
			}
			log.Println("Received task: ", task.ID, task.Url)
			err = handler(&task)
			if err != nil {
				log.Println("Error when processing task: ", task.ID, task.Url, err.Error())
				task.Error = append(task.Error, err.Error())
				newBody, e := json.Marshal(task)
				if e == nil {
					d.Body = newBody
				} else {
					log.Println(e)
				}
				retry(&d)
				return
			}
			newBody, e := json.Marshal(task)
			if e == nil {
				d.Body = newBody
			} else {
				log.Println(e)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err = MQChan.PublishWithContext(ctx, "page.events", "response", false, false, amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Headers: map[string]interface{}{
					"x-success": 1,
				},
				Body: d.Body,
			})
			if err != nil {
				log.Println(err)
				retry(&d)
				return
			}
			_ = d.Ack(false)
			log.Println("Acked task: ", task.ID, task.Url)
		}()
	}
}

func retry(d *amqp.Delivery) {
	d.Headers["retry-count"] = d.Headers["retry-count"].(int32) - 1
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := MQChan.PublishWithContext(ctx, "page.events.retry", "scrape", false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Headers:      d.Headers,
		Body:         d.Body,
	})
	if err != nil {
		_ = d.Reject(true)
	}
	_ = d.Ack(false)
}
