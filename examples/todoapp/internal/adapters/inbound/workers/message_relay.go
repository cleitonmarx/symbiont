package workers

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// MessageRelay is a runnable that processes outbox events and publishes them to Pub/Sub.
type MessageRelay struct {
	Uow      domain.UnitOfWork `resolve:""`
	Client   *pubsub.Client    `resolve:""`
	Logger   *log.Logger       `resolve:""`
	Interval time.Duration     `config:"FETCH_OUTBOX_INTERVAL" default:"500ms"`
}

// Run starts the periodic processing of outbox events.
func (op MessageRelay) Run(ctx context.Context) error {
	op.Logger.Println("OutboxPublisher: running...")
	ticker := time.NewTicker(op.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := op.processBatch(ctx)
			if err != nil {
				op.Logger.Printf("error processing batch: %v", err)
			}
		case <-ctx.Done():
			op.Logger.Println("OutboxPublisher: stopping...")
			return nil
		}
	}
}

// processBatch fetches a batch of pending outbox events, publishes them to Pub/Sub,
// and deletes or updates them based on the publishing result.
func (op MessageRelay) processBatch(ctx context.Context) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	err := op.Uow.Execute(spanCtx, func(uow domain.UnitOfWork) error {
		events, err := uow.Outbox().FetchPendingEvents(spanCtx, 100)
		if err != nil {
			return err
		}

		for _, event := range events {
			err := op.publishToPubSub(spanCtx, event)
			if err == nil {
				if err := uow.Outbox().DeleteEvent(spanCtx, event.ID); err != nil {
					return err
				}
			} else {
				if event.RetryCount+1 >= event.MaxRetries {
					if err := uow.Outbox().UpdateEvent(spanCtx, event.ID, "FAILED", event.RetryCount+1, err.Error()); err != nil {
						return err
					}
				} else {
					if err := uow.Outbox().UpdateEvent(spanCtx, event.ID, "PENDING", event.RetryCount+1, err.Error()); err != nil {
						return err
					}
				}
			}

		}
		return nil
	})
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	return nil
}

// publishToPubSub publishes a single outbox event to Pub/Sub.
func (op MessageRelay) publishToPubSub(ctx context.Context, event domain.OutboxEvent) error {
	spanCtx, span := tracing.Start(ctx,
		trace.WithAttributes(
			attribute.String("event_id", event.ID.String()),
			attribute.String("event_type", event.EventType),
			attribute.String("entity_id", event.EntityID.String()),
			attribute.String("topic", event.Topic),
		),
	)
	defer span.End()

	result := op.Client.Publisher(event.Topic).Publish(spanCtx, &pubsub.Message{
		Data: event.Payload,
		Attributes: map[string]string{
			"event_type": event.EventType,
			"todo_id":    event.EntityID.String(),
		},
	})

	_, err := result.Get(ctx)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}
	return nil
}
