package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/google/uuid"
)

// DeleteTodo defines the interface for the DeleteTodo use case.
type DeleteTodo interface {
	Execute(ctx context.Context, todoID uuid.UUID) error
}

// DeleteTodoImpl is the implementation of the DeleteTodo use case.
type DeleteTodoImpl struct {
	uow domain.UnitOfWork
}

// NewDeleteTodo creates a new instance of DeleteTodoImpl.
func NewDeleteTodo(uow domain.UnitOfWork) DeleteTodoImpl {
	return DeleteTodoImpl{
		uow: uow,
	}
}

// Execute deletes a todo item by its ID.
func (dti DeleteTodoImpl) Execute(ctx context.Context, todoID uuid.UUID) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	return dti.uow.Execute(spanCtx, func(uow domain.UnitOfWork) error {
		_, err := uow.Todo().GetTodo(spanCtx, todoID) // Ensure the todo exists
		if err != nil {
			return err
		}
		return uow.Todo().DeleteTodo(spanCtx, todoID)
	})
}

// InitDeleteTodo initializes the DeleteTodo use case.
type InitDeleteTodo struct {
	Uow domain.UnitOfWork `resolve:""`
}

// Initialize registers the DeleteTodo use case in the dependency container.
func (i InitDeleteTodo) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[DeleteTodo](NewDeleteTodo(i.Uow))
	return ctx, nil
}
