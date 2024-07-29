package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn         *amqp.Connection
	ch           *amqp.Channel
	exchangeName string
	exchangeType string
	logger       *slog.Logger
	config       Config
}

type Config struct {
	URL             string
	ExchangeName    string
	ExchangeType    string
	Logger          *slog.Logger
	ExchangeDurable bool
}

func NewRabbitMQ(cfg Config) (*RabbitMQ, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	return &RabbitMQ{
		conn:         conn,
		exchangeName: cfg.ExchangeName,
		exchangeType: cfg.ExchangeType,
		logger:       cfg.Logger,
		config:       cfg,
	}, nil
}

func (r *RabbitMQ) OpenChannel() error {
	if r.ch != nil {
		return nil
	}

	ch, err := r.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %w", err)
	}

	err = ch.ExchangeDeclare(
		r.exchangeName,
		r.exchangeType,
		r.config.ExchangeDurable,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		return fmt.Errorf("failed to declare an exchange: %w", err)
	}

	r.ch = ch
	r.logger.Info("Channel opened", "exchange", r.exchangeName)
	return nil
}

func (r *RabbitMQ) CloseChannel() error {
	if r.ch != nil {
		err := r.ch.Close()
		r.ch = nil
		if err != nil {
			return fmt.Errorf("error closing channel: %w", err)
		}
		r.logger.Info("Channel closed")
	}
	return nil
}

func (r *RabbitMQ) Publish(ctx context.Context, routingKey string, body []byte) error {
	if r.ch == nil {
		if err := r.OpenChannel(); err != nil {
			return err
		}
	}

	err := r.ch.PublishWithContext(
		ctx,
		r.exchangeName,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish a message: %w", err)
	}

	r.logger.Info("Message published", "exchange", r.exchangeName, "routingKey", routingKey)
	return nil
}

func (r *RabbitMQ) closeConnection() error {
	if r.conn != nil {
		err := r.conn.Close()
		if err != nil {
			return fmt.Errorf("error closing connection: %w", err)
		}
		r.logger.Info("Connection closed")
	}
	return nil
}

func (r *RabbitMQ) Close() error {
	if err := r.CloseChannel(); err != nil {
		return err
	}
	return r.closeConnection()
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := Config{
		URL:             "amqp://user:password@localhost:5672/",
		ExchangeName:    "exchange",
		ExchangeType:    "topic",
		Logger:          logger,
		ExchangeDurable: true,
	}

	rmq, err := NewRabbitMQ(cfg)
	if err != nil {
		logger.Error("Failed to initialize RabbitMQ", "error", err)
		panic(err)
	}
	defer rmq.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Publish a message, channel will be opened automatically if needed
	err = rmq.Publish(ctx, "example.route", []byte("Hello, RabbitMQ!"))
	if err != nil {
		logger.Error("Failed to publish message", "error", err)
	}

	// Close the channel explicitly if needed
	if err := rmq.CloseChannel(); err != nil {
		logger.Error("Failed to close channel", "error", err)
	}
}
