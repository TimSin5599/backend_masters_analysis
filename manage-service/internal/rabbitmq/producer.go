package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"manage-service/internal/domain/entity"
	"manage-service/pkg/metrics"

	amqp "github.com/rabbitmq/amqp091-go"
)

const QueueName = "document_extraction_queue"

type Producer struct {
	url  string
	conn *amqp.Connection
	ch   *amqp.Channel
	mu   sync.Mutex
}

func NewProducer(url string) (*Producer, error) {
	p := &Producer{url: url}
	if err := p.connect(); err != nil {
		return nil, err
	}
	return p, nil
}

// connect (re)establishes the AMQP connection and channel.
// Must be called with p.mu held or during construction.
func (p *Producer) connect() error {
	// Close stale resources before reconnecting.
	if p.ch != nil {
		p.ch.Close()
		p.ch = nil
	}
	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
	}

	conn, err := amqp.Dial(p.url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open a channel: %w", err)
	}

	args := amqp.Table{"x-max-priority": int32(10)}
	if _, err = ch.QueueDeclare(QueueName, true, false, false, false, args); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	p.conn = conn
	p.ch = ch
	return nil
}

func (p *Producer) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
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

	p.mu.Lock()
	defer p.mu.Unlock()

	if err = p.publishLocked(body, task.Priority); err != nil {
		// Connection likely died (Mac sleep). Attempt one reconnect and retry.
		log.Printf("[PRODUCER] Publish failed (%v), attempting reconnect...", err)
		if reconnErr := p.connect(); reconnErr != nil {
			return fmt.Errorf("failed to publish (reconnect also failed: %v): %w", reconnErr, err)
		}
		if err = p.publishLocked(body, task.Priority); err != nil {
			return fmt.Errorf("failed to publish after reconnect: %w", err)
		}
	}

	metrics.RabbitmqPublishedTotal.Inc()
	log.Printf(" [x] Sent document task %s with priority %d", task.ID, task.Priority)
	return nil
}

func (p *Producer) publishLocked(body []byte, priority int) error {
	return p.ch.Publish(
		"",        // exchange
		QueueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Priority:     uint8(priority),
			Body:         body,
		},
	)
}
