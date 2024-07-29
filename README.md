## Initializing the client
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
    panic(err)
}
defer rmq.Close()
```

## Publishing a message
Will open a channel, publish the message and close the channel.
```go
err := rmq.Publish(ctx, "example.route", []byte("Hello, RabbitMQ!"))
if err != nil {
    panic(err)
}
```

## Manually managing the channel
```go
ch, err := rmq.OpenChannel()
if err != nil {
    panic(err)
}
defer ch.Close()

err = rmq.PublishOnChannel(ctx, ch, "another.route", []byte("Another message"))
if err != nil {
    panic(err)
}

err = rmq.PublishOnChannel(ctx, ch, "yet.another.route", []byte("Yet another message"))
if err != nil {
    panic(err)
}
```

