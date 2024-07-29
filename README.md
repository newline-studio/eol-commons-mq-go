## Initialize connection
```go
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
```

## Publish a message
```go
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
```