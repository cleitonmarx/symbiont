package usecases

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/tracing"
	"github.com/google/uuid"
)

// CreateTodo defines the interface for the CreateTodo use case.
type CreateTodo interface {
	Execute(ctx context.Context, title string, dueDate time.Time) (domain.Todo, error)
}

// CreateTodoImpl is the implementation of the CreateTodo use case.
type CreateTodoImpl struct {
	todoRepo     domain.TodoRepository
	timeProvider domain.CurrentTimeProvider
	publisher    domain.TodoEventPublisher
	createUUID   func() uuid.UUID
}

// NewCreateTodoImpl creates a new instance of CreateTodoImpl.
func NewCreateTodoImpl(todoRepo domain.TodoRepository, timeProvider domain.CurrentTimeProvider, publisher domain.TodoEventPublisher) CreateTodoImpl {
	return CreateTodoImpl{
		todoRepo:     todoRepo,
		timeProvider: timeProvider,
		publisher:    publisher,
		createUUID:   uuid.New,
	}
}

// Execute creates a new todo item.
func (cti CreateTodoImpl) Execute(ctx context.Context, title string, dueDate time.Time) (domain.Todo, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	if len(title) < 3 || len(title) > 200 {
		err := domain.NewValidationErr("title must be between 3 and 200 characters")
		tracing.RecordErrorAndStatus(span, err)
		return domain.Todo{}, err
	}

	now := cti.timeProvider.Now()
	todo := domain.Todo{
		ID:          cti.createUUID(),
		Title:       title,
		Status:      domain.TodoStatus_OPEN,
		EmailStatus: domain.EmailStatus_PENDING,
		DueDate:     dueDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := cti.todoRepo.CreateTodo(spanCtx, todo)
	if tracing.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}

	err = cti.publisher.PublishEvent(spanCtx, domain.TodoEvent{
		Type:   domain.TodoEventType_TODO_CREATED,
		TodoID: todo.ID,
	})

	if tracing.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}

	return todo, nil
}

// InitCreateTodo initializes the CreateTodo use case and registers it in the dependency container.
type InitCreateTodo struct {
	Repo        domain.TodoRepository      `resolve:""`
	TimeService domain.CurrentTimeProvider `resolve:""`
	Publisher   domain.TodoEventPublisher  `resolve:""`
}

// Initialize initializes the CreateTodoImpl use case and registers it in the dependency container.
func (ict InitCreateTodo) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[CreateTodo](NewCreateTodoImpl(ict.Repo, ict.TimeService, ict.Publisher))
	return ctx, nil
}
