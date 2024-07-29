package main

import (
	"context"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn         *amqp.Connection
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

func (r *RabbitMQ) OpenChannel() (*amqp.Channel, error) {
	ch, err := r.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
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
		return nil, fmt.Errorf("failed to declare an exchange: %w", err)
	}

	r.logger.Info("Channel opened", "exchange", r.exchangeName)
	return ch, nil
}

func (r *RabbitMQ) Publish(ctx context.Context, routingKey string, body []byte) error {
	ch, err := r.OpenChannel()
	if err != nil {
		return err
	}
	defer ch.Close()

	err = r.PublishWithChannel(ctx, ch, routingKey, body)
	if err != nil {
		return fmt.Errorf("failed to publish a message: %w", err)
	}

	return nil
}

func (r *RabbitMQ) PublishWithChannel(ctx context.Context, ch *amqp.Channel, routingKey string, body []byte) error {
	err := ch.PublishWithContext(
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

func (r *RabbitMQ) Close() error {
	if r.conn != nil {
		err := r.conn.Close()
		if err != nil {
			return fmt.Errorf("error closing connection: %w", err)
		}
		r.logger.Info("Connection closed")
	}
	return nil
}
