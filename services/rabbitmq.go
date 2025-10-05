package services

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/resoul/avcompression/config"
	"github.com/resoul/avcompression/models"
	"github.com/sirupsen/logrus"
)

type RabbitMQService struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

func NewRabbitMQService(cfg config.RabbitMQConfig) (*RabbitMQService, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq failed: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel failed: %w", err)
	}

	q, err := ch.QueueDeclare(
		cfg.QueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare queue failed (queue=%s): %w", cfg.QueueName, err)
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
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("start consuming failed: %w", err)
	}

	forever := make(chan bool)

	go func() {
		for msg := range msgs {
			var job models.JobMessage
			if err := json.Unmarshal(msg.Body, &job); err != nil {
				logrus.WithError(err).Error("Failed to unmarshal job message")
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
