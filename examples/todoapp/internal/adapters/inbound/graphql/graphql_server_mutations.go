package graphql

import (
	"context"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql/gen"
	"github.com/google/uuid"
)

// Mutation returns MutationResolver implementation.
func (s *TodoGraphQLServer) Mutation() gen.MutationResolver { return s }

// UpdateTodo is the resolver for the updateTodo field.
func (s *TodoGraphQLServer) UpdateTodo(ctx context.Context, params gen.UpdateTodoParams) (*gen.Todo, error) {
	return nil, nil
}

// DeleteTodo is the resolver for the deleteTodo field.
func (s *TodoGraphQLServer) DeleteTodo(ctx context.Context, id uuid.UUID) (bool, error) {
	return false, nil
}
