package usecases

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/tracing"
	"github.com/google/uuid"
)

type UpdateTodo interface {
	Execute(ctx context.Context, id uuid.UUID, title *string, status *domain.TodoStatus, dueDate *time.Time) (domain.Todo, error)
}

// UpdateTodoImpl is the implementation of the UpdateTodo use case.
type UpdateTodoImpl struct {
	todoRepo    domain.TodoRepository     `resolve:""`
	timeService domain.TimeService        `resolve:""`
	publisher   domain.TodoEventPublisher `resolve:""`
}

// NewUpdateTodoImpl creates a new instance of UpdateTodoImpl.
func NewUpdateTodoImpl(todoRepo domain.TodoRepository, timeService domain.TimeService, publisher domain.TodoEventPublisher) UpdateTodoImpl {
	return UpdateTodoImpl{
		todoRepo:    todoRepo,
		timeService: timeService,
		publisher:   publisher,
	}
}

// Execute updates an existing todo item identified by id with the provided title and/or status.
func (uti UpdateTodoImpl) Execute(ctx context.Context, id uuid.UUID, title *string, status *domain.TodoStatus, dueDate *time.Time) (domain.Todo, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	todo, err := uti.todoRepo.GetTodo(spanCtx, id)
	if tracing.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}

	if title != nil {
		todo.Title = *title
	}

	if status != nil {
		todo.Status = *status
	}
	if dueDate != nil {
		todo.DueDate = *dueDate
	}

	todo.UpdatedAt = uti.timeService.Now()

	err = uti.todoRepo.UpdateTodo(spanCtx, todo)
	if tracing.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}

	err = uti.publisher.PublishEvent(spanCtx, domain.TodoEvent{
		Type:   domain.TodoEventType_TODO_UPDATED,
		TodoID: todo.ID,
	})
	if tracing.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}

	return todo, nil
}

// InitUpdateTodo initializes the UpdateTodo use case and registers it in the dependency container.
type InitUpdateTodo struct {
	Repo        domain.TodoRepository     `resolve:""`
	TimeService domain.TimeService        `resolve:""`
	Publisher   domain.TodoEventPublisher `resolve:""`
}

// Initialize initializes the UpdateTodoImpl use case.
func (iut InitUpdateTodo) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[UpdateTodo](NewUpdateTodoImpl(iut.Repo, iut.TimeService, iut.Publisher))
	return ctx, nil
}
