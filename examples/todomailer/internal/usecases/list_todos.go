package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/tracing"
)

// ListTodos defines the interface for the ListTodos use case.
type ListTodos interface {
	Query(ctx context.Context, page int, pageSize int, opts ...domain.ListTodoOptions) ([]domain.Todo, bool, error)
}

// ListTodosImpl is the implementation of the ListTodos use case.
type ListTodosImpl struct {
	Repo domain.Repository `resolve:""`
}

func NewListTodosImpl(repo domain.Repository) ListTodosImpl {
	return ListTodosImpl{
		Repo: repo,
	}
}

// Query retrieves a list of todo items with pagination support.
func (lti ListTodosImpl) Query(ctx context.Context, page int, pageSize int, opts ...domain.ListTodoOptions) ([]domain.Todo, bool, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	todos, hasMore, err := lti.Repo.ListTodos(spanCtx, page, pageSize, opts...)
	if tracing.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}
	return todos, hasMore, nil
}

type InitListTodos struct {
	Repo domain.Repository `resolve:""`
}

// Initialize initializes the ListTodosImpl use case.
func (ilt *InitListTodos) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ListTodos](NewListTodosImpl(ilt.Repo))
	return ctx, nil
}
