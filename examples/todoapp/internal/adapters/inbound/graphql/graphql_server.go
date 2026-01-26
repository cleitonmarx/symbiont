package graphql

// THIS CODE WILL BE UPDATED WITH SCHEMA CHANGES. PREVIOUS IMPLEMENTATION FOR SCHEMA CHANGES WILL BE KEPT IN THE COMMENT SECTION. IMPLEMENTATION FOR UNCHANGED SCHEMA WILL BE KEPT.

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql/graph"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type TodoGraphQLServer struct {
	Logger           *log.Logger        `resolve:""`
	ListTodosUsecase usecases.ListTodos `resolve:""`
	Port             int                `config:"GRAPHQL_SERVER_PORT" default:"8085"`
}

// MarkTodosDone is the resolver for the markTodosDone field.
func (s *TodoGraphQLServer) MarkTodosDone(ctx context.Context, ids []*uuid.UUID) (bool, error) {
	return false, nil
}

// DeleteTodos is the resolver for the deleteTodos field.
func (s *TodoGraphQLServer) DeleteTodos(ctx context.Context, ids []*uuid.UUID) (bool, error) {
	return false, nil
}

// ListTodos is the resolver for the listTodos field.
func (s *TodoGraphQLServer) ListTodos(ctx context.Context, status *graph.TodoStatus, page int, pageSize int) (*graph.TodoPage, error) {
	var options []domain.ListTodoOptions
	if status != nil {
		options = append(options, domain.WithStatus(domain.TodoStatus(*status)))
	}
	todos, hasMore, err := s.ListTodosUsecase.Query(ctx, page, pageSize, options...)
	if err != nil {
		return nil, err
	}

	todoPage := graph.TodoPage{
		Items: make([]*graph.Todo, len(todos)),
		Page:  page,
	}

	for i, t := range todos {
		todoPage.Items[i] = &graph.Todo{
			ID:        t.ID,
			Title:     t.Title,
			Status:    graph.TodoStatus(t.Status),
			DueDate:   t.DueDate.Format("2006-01-02"),
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		}
	}

	if hasMore {
		todoPage.NextPage = common.Ptr(page + 1)
	}
	if page > 1 {
		todoPage.PreviousPage = common.Ptr(page - 1)
	}

	return &todoPage, nil
}

// Mutation returns MutationResolver implementation.
func (s *TodoGraphQLServer) Mutation() graph.MutationResolver { return s }

// Query returns QueryResolver implementation.
func (s *TodoGraphQLServer) Query() graph.QueryResolver { return s }

func (s *TodoGraphQLServer) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	h := handler.New(
		graph.NewExecutableSchema(graph.Config{Resolvers: s}),
	)
	h.AddTransport(transport.POST{})
	h.AddTransport(transport.GET{})

	mux.Handle("/query", otelhttp.NewHandler(
		h,
		"",
		otelhttp.WithSpanNameFormatter(tracing.SpanNameFormatter),
	))

	mux.Handle("/", playground.Handler("TodoApp GraphQL", "/query"))

	svr := &http.Server{
		Handler: mux,
		Addr:    fmt.Sprintf(":%d", s.Port),
	}

	errCh := make(chan error, 1)
	go func() {
		s.Logger.Printf("TodoGraphQLServer: Listening on port %d", s.Port)
		errCh <- svr.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		s.Logger.Print("TodoGraphQLServer: Shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return svr.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
