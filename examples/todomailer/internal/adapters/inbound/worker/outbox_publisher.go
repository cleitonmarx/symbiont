package worker

import (
	"context"
	"database/sql"
	"log"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/tracing"
	"github.com/google/uuid"
)

// OutboxPublisher periodically fetches outbox events from the database and publishes them to Pub/Sub.
type OutboxPublisher struct {
	DB       *sql.DB        `resolve:""`
	Client   *pubsub.Client `resolve:""`
	Logger   *log.Logger    `resolve:""`
	Interval time.Duration  `config:"FETCH_OUTBOX_INTERVAL" default:"500ms"`
	TopicID  string         `config:"PUBSUB_TOPIC_ID" default:"todo-events"`
}

// Run starts the periodic processing of outbox events.
func (op OutboxPublisher) Run(ctx context.Context) error {
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
func (op OutboxPublisher) processBatch(ctx context.Context) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	tx, err := op.DB.BeginTx(spanCtx, nil)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			op.Logger.Printf("error rolling back transaction: %v", err)
		}
	}()

	events, err := fetchPendingOutboxEvents(spanCtx, tx, 100)
	if err != nil {
		return err
	}

	for _, event := range events {
		result := op.Client.Publisher(op.TopicID).Publish(ctx, &pubsub.Message{
			Data: event.Payload,
			Attributes: map[string]string{
				"event_type": "TODO_EVENT",
				"todo_id":    event.TodoID.String(),
			},
		})

		_, err := result.Get(ctx)
		if err == nil {
			if err := deleteOutboxEvent(spanCtx, tx, event.ID); err != nil {
				return err
			}
		} else {
			if event.RetryCount+1 >= event.MaxRetries {
				if err := updateOutboxEvent(spanCtx, tx, event.ID, "FAILED", event.RetryCount+1, err.Error()); err != nil {
					return err
				}
			} else {
				if err := updateOutboxEvent(spanCtx, tx, event.ID, "PENDING", event.RetryCount+1, err.Error()); err != nil {
					return err
				}
			}
		}
	}

	err = tx.Commit()
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}
	return nil
}

// outboxItem represents a row in the outbox_events table.
type outboxItem struct {
	ID         uuid.UUID
	TodoID     uuid.UUID
	Payload    []byte
	RetryCount int
	MaxRetries int
	LastError  sql.NullString
	CreatedAt  time.Time
}

// fetchPendingOutboxEvents retrieves a batch of pending outbox events from the database.
func fetchPendingOutboxEvents(ctx context.Context, tx *sql.Tx, limit int) ([]outboxItem, error) {
	rows, err := squirrel.
		StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(tx).
		Select(
			"id",
			"todo_id",
			"payload",
			"retry_count",
			"max_retries",
			"last_error",
			"created_at",
		).
		From("outbox_events").
		Where(squirrel.Eq{"status": "PENDING"}).
		OrderBy("created_at ASC").
		Limit(uint64(limit)).
		Suffix("FOR UPDATE SKIP LOCKED").
		QueryContext(ctx)

	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var events []outboxItem
	for rows.Next() {
		var oe outboxItem
		var payloadBytes []byte
		err := rows.Scan(
			&oe.ID,
			&oe.TodoID,
			&payloadBytes,
			&oe.RetryCount,
			&oe.MaxRetries,
			&oe.LastError,
			&oe.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		events = append(events, oe)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// updateOutboxEvent updates the status, retry count, and last error of an outbox event.
func updateOutboxEvent(ctx context.Context, tx *sql.Tx, eventID uuid.UUID, status string, retryCount int, lastError string) error {
	_, err := squirrel.
		StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(tx).
		Update("outbox_events").
		Set("status", status).
		Set("retry_count", retryCount).
		Set("last_error", lastError).
		Where(squirrel.Eq{"id": eventID}).
		ExecContext(ctx)

	return err
}

// deleteOutboxEvent deletes an outbox event from the database.
func deleteOutboxEvent(ctx context.Context, tx *sql.Tx, eventID uuid.UUID) error {
	_, err := squirrel.
		StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(tx).
		Delete("outbox_events").
		Where(squirrel.Eq{"id": eventID}).
		ExecContext(ctx)

	return err
}
