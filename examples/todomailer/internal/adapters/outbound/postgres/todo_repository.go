package postgres

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/tracing"
)

// TodoRepository is an in-memory implementation of domain.Repository for Todos.
type TodoRepository struct {
	items map[uuid.UUID]domain.Todo
}

// NewTodoRepository creates a new instance of TodoRepository.
func NewTodoRepository() *TodoRepository {
	return &TodoRepository{
		items: make(map[uuid.UUID]domain.Todo),
	}
}

// ListTodos lists todos with pagination and optional filters.
func (tr *TodoRepository) ListTodos(ctx context.Context, page int, pageSize int, opts ...domain.ListTodoOptions) ([]domain.Todo, bool, error) {
	_, span := tracing.Start(ctx, trace.WithAttributes(
		attribute.Int("page", page),
		attribute.Int("pageSize", pageSize),
	))
	defer span.End()

	params := &domain.ListTodosParams{}
	for _, opt := range opts {
		opt(params)
	}
	var todos []domain.Todo
	for _, todo := range tr.items {
		if params.Status != nil && todo.Status != *params.Status {
			continue
		}
		if params.EmailStatus != nil && todo.EmailStatus != *params.EmailStatus {
			continue
		}
		todos = append(todos, todo)
	}

	sort.Slice(todos, func(i, j int) bool {
		return todos[i].CreatedAt.Before(todos[j].CreatedAt)
	})

	// pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 1
	}
	offset := (page - 1) * pageSize
	if offset >= len(todos) {
		return []domain.Todo{}, false, nil
	}
	end := offset + pageSize
	if end > len(todos) {
		end = len(todos)
	}
	hasMore := end < len(todos)

	return todos[offset:end], hasMore, nil
}

// CreateTodo creates a new todo.
func (tr *TodoRepository) CreateTodo(ctx context.Context, todo domain.Todo) error {
	_, span := tracing.Start(ctx)
	defer span.End()

	tr.items[todo.Id] = todo
	return nil
}

// UpdateTodo updates an existing todo.
func (tr *TodoRepository) UpdateTodo(ctx context.Context, todo domain.Todo) error {
	_, span := tracing.Start(ctx)
	defer span.End()

	tr.items[todo.Id] = todo
	return nil
}

// GetTodo retrieves a todo by its ID.
func (tr *TodoRepository) GetTodo(ctx context.Context, id uuid.UUID) (domain.Todo, error) {
	_, span := tracing.Start(ctx)
	defer span.End()

	todo, exists := tr.items[id]
	if !exists {
		return domain.Todo{}, domain.NewValidationErr(fmt.Sprintf("todo with id %s not found", id))
	}
	return todo, nil
}

// InitTodoRepository is a Symbiont initializer for TodoRepository.
type InitTodoRepository struct {
}

// Initialize registers the TodoRepository in the dependency container.
func (tr *InitTodoRepository) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.Repository](NewTodoRepository())
	return ctx, nil
}
