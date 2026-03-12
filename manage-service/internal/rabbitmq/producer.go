package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"

	"manage-service/internal/entity"

	amqp "github.com/rabbitmq/amqp091-go"
)

const QueueName = "document_extraction_queue"

type Producer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewProducer(url string) (*Producer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Declare a queue with max priority 10
	args := amqp.Table{
		"x-max-priority": int32(10),
	}

	_, err = ch.QueueDeclare(
		QueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		args,  // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	return &Producer{
		conn: conn,
		ch:   ch,
	}, nil
}

func (p *Producer) Close() {
	if p.ch != nil {
		p.ch.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}

func (p *Producer) PublishTask(task entity.DocumentQueueTask) error {
	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	err = p.ch.Publish(
		"",        // exchange
		QueueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Priority:     uint8(task.Priority),
			Body:         body,
		})
	if err != nil {
		return fmt.Errorf("failed to publish a message: %w", err)
	}

	log.Printf(" [x] Sent document task %s with priority %d", task.ID, task.Priority)
	return nil
}
