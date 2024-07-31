package commonsMq

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func MiddlewareRecover(logger *slog.Logger) EventSubscriptionMiddleware {
	return func(next EventSubscription) EventSubscription {
		return func(ctx context.Context, delivery amqp.Delivery) error {
			defer func() {
				if r := recover(); r != nil {
					logger.Error(
						"recovered from panic",
						"error", r,
						"stack", string(debug.Stack()),
					)
				}
			}()
			return next(ctx, delivery)
		}
	}
}

func MiddlewareLogger(logger *slog.Logger) EventSubscriptionMiddleware {
	return func(next EventSubscription) EventSubscription {
		return func(ctx context.Context, delivery amqp.Delivery) error {
			start := time.Now()
			err := next(ctx, delivery)
			logger.Info(
				"process event",
				"routingKey", delivery.RoutingKey,
				"error", err,
				"duration", time.Since(start).String(),
			)
			return err
		}
	}
}
