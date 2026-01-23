package domain

import "context"

// UnitOfWork represents a unit of work for managing repositories and transactions.
type UnitOfWork interface {
	Todo() TodoRepository
	Publisher() TodoEventPublisher
	Outbox() OutboxRepository
	// Execute runs a function within the context of a unit of work.
	Execute(ctx context.Context, fn func(uow UnitOfWork) error) error
}
