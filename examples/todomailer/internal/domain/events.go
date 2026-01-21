package domain

import (
	"context"

	"github.com/google/uuid"
)

type TodoEventType string

const (
	// TodoEventType_TODO_CREATED represents the event when a todo item is created.
	TodoEventType_TODO_CREATED TodoEventType = "TODO.CREATED"
	// TodoEventType_TODO_UPDATED represents the event when a todo item is updated.
	TodoEventType_TODO_UPDATED TodoEventType = "TODO.UPDATED"
)

// TodoEvent represents a domain event in the system.
type TodoEvent struct {
	Type   TodoEventType
	TodoID uuid.UUID
}

// TodoEventPublisher defines the interface for publishing todo events.
type TodoEventPublisher interface {
	PublishEvent(ctx context.Context, event TodoEvent) error
}
