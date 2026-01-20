package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
)

type TodoRepository struct {
	items map[uuid.UUID]domain.Todo
}

func NewTodoRepository() *TodoRepository {
	return &TodoRepository{
		items: make(map[uuid.UUID]domain.Todo),
	}
}

func (tr *TodoRepository) ListTodos(ctx context.Context, page int, pageSize int, opts ...domain.ListTodoOptions) ([]domain.Todo, bool, error) {
	params := &domain.ListTodosParams{}
	for _, opt := range opts {
		opt(params)
	}
	var todos []domain.Todo
	for _, todo := range tr.items {
		if params.Status != nil && todo.Status != *params.Status {
			continue
		}
		todos = append(todos, todo)
	}

	if len(todos) > pageSize {
		return todos[:pageSize], true, nil
	}

	return todos, false, nil
}

func (tr *TodoRepository) CreateTodo(ctx context.Context, todo domain.Todo) error {
	tr.items[todo.Id] = todo
	return nil
}

func (tr *TodoRepository) UpdateTodo(ctx context.Context, todo domain.Todo) error {
	tr.items[todo.Id] = todo
	return nil
}

func (tr *TodoRepository) GetTodo(ctx context.Context, id uuid.UUID) (domain.Todo, error) {
	todo, exists := tr.items[id]
	if !exists {
		return domain.Todo{}, domain.NewValidationErr(fmt.Sprintf("todo with id %s not found", id))
	}
	return todo, nil
}

type InitTodoRepository struct {
}

func (tr *InitTodoRepository) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.Repository](NewTodoRepository())
	return ctx, nil
}
