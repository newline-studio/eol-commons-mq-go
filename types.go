package commonsMq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Connection interface {
	Publisher(config PublisherConfig) Publisher
	Subscriber(config SubscriberConfig) Subscriber
	Close() error
}

type EventPublish func(ctx context.Context, routingKey string, body any, mandatory bool, immediate bool) error
type ChannelClose func() error

type Publisher interface {
	Publish(ctx context.Context, routingKey string, body any, mandatory bool, immediate bool) error
	PublishMultiple() (EventPublish, ChannelClose, error)
}

type EventSubscription func(ctx context.Context, delivery amqp.Delivery) error
type EventSubscriptionMiddleware func(ctx context.Context, delivery amqp.Delivery, next EventSubscription) error

type Subscriber interface {
	Subscribe(routingKey string, handler EventSubscription, noWait bool, args map[string]interface{}) error
	Listen(ctx context.Context, middleware EventSubscriptionMiddleware, autoAck bool, exclusive bool, noLocal bool, noWait bool, args map[string]any) error
}

type SubscriberConfig struct {
	ExchangeConfig ExchangeConfig
	QueueConfig    QueueConfig
}

type PublisherConfig struct {
	ExchangeConfig ExchangeConfig
}

type ExchangeConfig struct {
	Name       string
	Type       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       map[string]interface{}
}

type QueueConfig struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       map[string]interface{}
}
