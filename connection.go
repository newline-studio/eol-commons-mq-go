package commonsMq

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

func NewConnection(url string) (Connection, error) {
	manager := &amqpConnection{
		conn:          nil,
		connectionUrl: url,
	}
	err := manager.reestablishConnection()
	return manager, err
}

type amqpConnection struct {
	conn          *amqp.Connection
	connectionUrl string
	sync          sync.Mutex
}

func (a *amqpConnection) reestablishConnection() error {
	a.sync.Lock()
	defer a.sync.Unlock()
	if a.conn != nil && !a.conn.IsClosed() {
		return nil
	}
	conn, err := amqp.Dial(a.connectionUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to message queue: %w", err)
	}
	a.conn = conn
	return nil
}

func (a *amqpConnection) Publisher(config PublisherConfig) Publisher {
	return &amqpPublisher{
		connection: a,
		config:     config,
	}
}

func (a *amqpConnection) Subscriber(config SubscriberConfig) Subscriber {
	return &amqpSubscriber{
		connection: a,
		config:     config,
	}
}

func (a *amqpConnection) Close() error {
	if a.conn == nil {
		return fmt.Errorf("connection is not established")
	}
	if a.conn.IsClosed() {
		return nil
	}
	err := a.conn.Close()
	if err != nil {
		return fmt.Errorf("error closing connection: %w", err)
	}
	return nil
}

func (a *amqpConnection) OpenChannel(config ExchangeConfig) (*amqp.Channel, error) {
	if err := a.reestablishConnection(); err != nil {
		return nil, fmt.Errorf("unable to reestablish connection: %w", err)
	}
	ch, err := a.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	err = ch.ExchangeDeclare(
		config.Name,
		config.Type,
		config.Durable,
		config.AutoDelete,
		config.Internal,
		config.NoWait,
		config.Args,
	)
	if err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("failed to declare an exchange: %w", err)
	}
	return ch, nil
}

type amqpHandler struct {
	handler    EventSubscription
	routingKey string
	exp        *regexp.Regexp
	noWait     bool
	args       map[string]interface{}
}

type amqpSubscriber struct {
	connection *amqpConnection
	config     SubscriberConfig
	lock       sync.Mutex
	handlers   []amqpHandler
}

func (a *amqpSubscriber) Subscribe(routingKey string, handler EventSubscription, noWait bool, args map[string]interface{}) error {
	// Prepare RoutingKey pattern matcher
	words := strings.Split(routingKey, ".")
	for i, word := range words {
		switch word {
		case "#":
			words[i] = strings.ReplaceAll(word, "#", "(.*?)")
		case "*":
			words[i] = strings.ReplaceAll(word, "*", "([^\\.]*?)")
		}
	}
	pattern := "^" + strings.Join(words, "\\.") + "$"
	exp, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("failed to convert routing key pattern to regular expression: %w", err)
	}

	// Append handler to the list
	a.lock.Lock()
	a.handlers = append(a.handlers, amqpHandler{
		handler:    handler,
		routingKey: routingKey,
		exp:        exp,
		noWait:     noWait,
		args:       args,
	})
	a.lock.Unlock()
	return nil
}

func (a *amqpSubscriber) Listen(ctx context.Context, middleware EventSubscriptionMiddleware, autoAck bool, exclusive bool, noLocal bool, noWait bool, args map[string]any) error {
	if err := a.connection.reestablishConnection(); err != nil {
		return fmt.Errorf("unable to reestablish connection: %w", err)
	}

	connection, err := a.connection.OpenChannel(a.config.ExchangeConfig)
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	queue, err := connection.QueueDeclare(
		a.config.QueueConfig.Name,
		a.config.QueueConfig.Durable,
		a.config.QueueConfig.AutoDelete,
		a.config.QueueConfig.Exclusive,
		a.config.QueueConfig.NoWait,
		a.config.QueueConfig.Args,
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	for _, handler := range a.handlers {
		if err := connection.QueueBind(
			queue.Name,
			handler.routingKey,
			a.config.ExchangeConfig.Name,
			handler.noWait,
			handler.args,
		); err != nil {
			return fmt.Errorf("failed to bind a queue: %w", err)
		}
	}

	messages, err := connection.Consume(
		queue.Name,
		"",
		autoAck,
		exclusive,
		noLocal,
		noWait,
		args,
	)
	if err != nil {
		return fmt.Errorf("failed to create message consumer: %w", err)
	}

	// Read messages from the channel
	for delivery := range messages {
		for _, handler := range a.handlers {
			if handler.exp.MatchString(delivery.RoutingKey) {
				if middleware != nil {
					_ = middleware(handler.handler)(ctx, delivery)
				} else {
					_ = handler.handler(ctx, delivery)
				}
			}
		}
	}
	return nil
}

type amqpPublisher struct {
	connection *amqpConnection
	config     PublisherConfig
}

func (a *amqpPublisher) Publish(ctx context.Context, routingKey string, body any, mandatory bool, immediate bool) error {
	if err := a.connection.reestablishConnection(); err != nil {
		return fmt.Errorf("unable to reestablish connection: %w", err)
	}
	publish, closer, err := a.PublishMultiple()
	if err != nil {
		return fmt.Errorf("failed to prepare publishing channel: %w", err)
	}
	defer closer()
	if err := publish(ctx, routingKey, body, mandatory, immediate); err != nil {
		return fmt.Errorf("failed to publish a message: %w", err)
	}
	return nil
}

func (a *amqpPublisher) PublishMultiple() (EventPublish, ChannelClose, error) {
	if err := a.connection.reestablishConnection(); err != nil {
		return nil, nil, fmt.Errorf("unable to reestablish connection: %w", err)
	}
	channel, err := a.connection.OpenChannel(a.config.ExchangeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open channel: %w", err)
	}

	publisher := func(ctx context.Context, routingKey string, body any, mandatory bool, immediate bool) error {
		var bodyString []byte
		switch body.(type) {
		case string:
			bodyString = []byte(body.(string))
		case []byte:
			bodyString = body.([]byte)
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("failed to marshal body: %w", err)
			}
			bodyString = jsonData
		}
		if err := channel.PublishWithContext(
			ctx,
			a.config.ExchangeConfig.Name,
			routingKey,
			mandatory,
			immediate,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        bodyString,
			},
		); err != nil {
			return fmt.Errorf("failed to publish a message: %w", err)
		}
		return nil
	}

	return publisher, channel.Close, nil
}
