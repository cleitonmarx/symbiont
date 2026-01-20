package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/tracing"
	"github.com/google/uuid"
)

type UpdateTodo interface {
	Execute(ctx context.Context, id uuid.UUID, title *string, status *domain.TodoStatus) (domain.Todo, error)
}

// UpdateTodoImpl is the implementation of the UpdateTodo use case.
type UpdateTodoImpl struct {
	repo        domain.Repository  `resolve:""`
	timeService domain.TimeService `resolve:""`
}

func NewUpdateTodoImpl(repo domain.Repository, timeService domain.TimeService) UpdateTodoImpl {
	return UpdateTodoImpl{
		repo:        repo,
		timeService: timeService,
	}
}

// Execute updates an existing todo item identified by id with the provided title and/or status.
func (uti UpdateTodoImpl) Execute(ctx context.Context, id uuid.UUID, title *string, status *domain.TodoStatus) (domain.Todo, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	todo, err := uti.repo.GetTodo(spanCtx, id)
	if tracing.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}

	if title != nil {
		todo.Title = *title
	}

	if status != nil {
		todo.Status = *status
	}

	todo.UpdatedAt = uti.timeService.Now()

	err = uti.repo.UpdateTodo(spanCtx, todo)
	if tracing.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}

	return todo, nil
}

type InitUpdateTodo struct {
	Repo        domain.Repository  `resolve:""`
	TimeService domain.TimeService `resolve:""`
}

// Initialize initializes the UpdateTodoImpl use case.
func (iut *InitUpdateTodo) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[UpdateTodo](NewUpdateTodoImpl(iut.Repo, iut.TimeService))
	return ctx, nil
}
