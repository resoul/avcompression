package services

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/resoul/avcompression/config"
	"github.com/resoul/avcompression/models"
)

type RabbitMQService struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

func NewRabbitMQService(cfg config.RabbitMQConfig) (*RabbitMQService, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	q, err := ch.QueueDeclare(
		cfg.QueueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &RabbitMQService{
		conn:    conn,
		channel: ch,
		queue:   q,
	}, nil
}

func (s *RabbitMQService) Consume(handler func(models.JobMessage)) error {
	msgs, err := s.channel.Consume(
		s.queue.Name,
		"",    // consumer tag
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	forever := make(chan bool)

	go func() {
		for msg := range msgs {
			var job models.JobMessage
			if err := json.Unmarshal(msg.Body, &job); err != nil {
				log.Printf("❌ failed to parse job: %v", err)
				continue
			}
			go handler(job)
		}
	}()

	<-forever
	return nil
}

func (s *RabbitMQService) Close() {
	if s.channel != nil {
		s.channel.Close()
	}
	if s.conn != nil {
		s.conn.Close()
	}
}
