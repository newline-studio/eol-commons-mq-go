## Initializing the client

```go
connection, err := commonsMq.NewConnection(os.Getenv("RABBIT_MQ_URL"))
if err != nil {
    panic(err)
}
defer connection.Close()
```

## Publishing messages
```go
// Use the connection to create the Publisher Interface
// which can safely be used to publish messages anywhere in the application.
publisher := connection.Publisher(commonsMq.PublisherConfig{
    ExchangeConfig: commonsMq.ExchangeConfig{
        Name: "system",
        Type: "topic",
    },
})

// Will open a channel, publish the message and close the channel.
_ = publisher.Publish(ctx, "foo.bar.baz", "body", false, false)

// Manual process, allowing publishing multiple messages on the same channel.
publish, closeChannel, err := publisher.PublishMultiple()
if err != nil {
panic(err)
}
defer closeChannel()
_ = publish(ctx, "foo.bar.baz", "body", false, false)
_ = publish(ctx, "foo.bar.baz", "body", false, false)
```

## Processing messages
```go
// Use the connection to create the Subscriber Interface
// which enables processing messages from the message queue
subscriber := connection.Subscriber(commonsMq.SubscriberConfig{
    ExchangeConfig: commonsMq.ExchangeConfig{
        Name: "system",
        Type: "topic",
    },
    QueueConfig: commonsMq.QueueConfig{
        Name: "queue-name",
    },
})

// Create as many binds as desired
_ = subscriber.Subscribe("foo.bar.baz", func(ctx context.Context, delivery amqp.Delivery) error {
    fmt.Println("1 -> ", string(delivery.Body))
    return nil
}, false, nil)

_ = subscriber.Subscribe("foo.*.baz", func(ctx context.Context, delivery amqp.Delivery) error {
    fmt.Println("2 -> ", string(delivery.Body))
    return nil
}, false, nil)

_ = subscriber.Subscribe("#.baz", func(ctx context.Context, delivery amqp.Delivery) error {
    fmt.Println("3 -> ", string(delivery.Body))
    return nil
}, false, nil)

// Global delivery middleware
middleware := func(next commonsMq.EventSubscription) commonsMq.EventSubscription {
    return func(ctx context.Context, delivery amqp.Delivery) error {
        fmt.Println("MIDDLEWARE")
        return next(ctx, delivery)
    }
}

// Will listen for messages forever
err := subscriber.Listen(ctx, middleware, true, false, false, false, nil)
if err != nil {
    panic(err)
}
```

