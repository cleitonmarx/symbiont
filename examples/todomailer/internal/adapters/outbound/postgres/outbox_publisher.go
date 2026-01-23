package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/tracing"
	"github.com/google/uuid"
)

type OutboxPublisher struct {
	sb squirrel.StatementBuilderType
}

func NewOutboxPublisher(br squirrel.BaseRunner) OutboxPublisher {
	return OutboxPublisher{
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(br),
	}
}

func (op OutboxPublisher) PublishEvent(ctx context.Context, event domain.TodoEvent) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	// Marshal the content to JSON
	contentJSON, err := json.Marshal(event)
	if tracing.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to marshal summary content: %w", err)
	}

	_, err = op.sb.Insert("outbox_events").
		Columns(
			"id",
			"todo_id",
			"payload",
			"retry_count",
			"max_retries",
			"last_error",
			"created_at",
		).
		Values(
			uuid.New(),
			event.TodoID,
			contentJSON,
			0,
			3,
			nil,
			event.CreatedAt,
		).
		ExecContext(spanCtx)

	if tracing.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}

	return nil
}
