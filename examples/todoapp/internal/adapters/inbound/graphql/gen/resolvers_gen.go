package gen

// THIS CODE WILL BE UPDATED WITH SCHEMA CHANGES. PREVIOUS IMPLEMENTATION FOR SCHEMA CHANGES WILL BE KEPT IN THE COMMENT SECTION. IMPLEMENTATION FOR UNCHANGED SCHEMA WILL BE KEPT.

import (
	"context"

	"github.com/google/uuid"
)

type Resolver struct{}

// MarkTodosDone is the resolver for the markTodosDone field.
func (r *mutationResolver) MarkTodosDone(ctx context.Context, ids []*uuid.UUID) (bool, error) {
	panic("not implemented")
}

// DeleteTodos is the resolver for the deleteTodos field.
func (r *mutationResolver) DeleteTodos(ctx context.Context, ids []*uuid.UUID) (bool, error) {
	panic("not implemented")
}

// ListTodos is the resolver for the listTodos field.
func (r *queryResolver) ListTodos(ctx context.Context, status *TodoStatus, page int, pageSize int) (*TodoPage, error) {
	panic("not implemented")
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
