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
	repo        domain.Repository
	timeService domain.TimeService
	createUUID  func() uuid.UUID
}

// NewCreateTodoImpl creates a new instance of CreateTodoImpl.
func NewCreateTodoImpl(repo domain.Repository, timeService domain.TimeService) CreateTodoImpl {
	return CreateTodoImpl{
		repo:        repo,
		timeService: timeService,
		createUUID:  uuid.New,
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

	now := cti.timeService.Now()
	todo := domain.Todo{
		Id:          cti.createUUID(),
		Title:       title,
		Status:      domain.TodoStatus_OPEN,
		EmailStatus: domain.EmailStatus_PENDING,
		DueDate:     dueDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := cti.repo.CreateTodo(spanCtx, todo)
	if tracing.RecordErrorAndStatus(span, err) {
		return domain.Todo{}, err
	}

	return todo, nil
}

// InitCreateTodo initializes the CreateTodo use case and registers it in the dependency container.
type InitCreateTodo struct {
	Repo        domain.Repository  `resolve:""`
	TimeService domain.TimeService `resolve:""`
}

// Initialize initializes the CreateTodoImpl use case and registers it in the dependency container.
func (ict *InitCreateTodo) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[CreateTodo](NewCreateTodoImpl(ict.Repo, ict.TimeService))
	return ctx, nil
}
