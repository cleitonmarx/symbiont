package http

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/usecases"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// TodoMailerApp is the HTTP server adapter for the TodoMailer application.
//
// It implements the OpenAPI-generated ServerInterface and serves both the REST API
// endpoints and the embedded web application static files. The server is instrumented
// with OpenTelemetry for distributed tracing and configured via environment variables
// or configuration providers through the symbiont framework.
//
// Dependencies are automatically resolved and injected at initialization time.
type TodoMailerApp struct {
	ServerInterface
	Port              int                 `config:"HTTP_PORT" default:"8080"`
	Logger            *log.Logger         `resolve:""`
	ListTodosUseCase  usecases.ListTodos  `resolve:""`
	CreateTodoUseCase usecases.CreateTodo `resolve:""`
	UpdateTodoUseCase usecases.UpdateTodo `resolve:""`
}

func (api *TodoMailerApp) ListTodos(w http.ResponseWriter, r *http.Request, params ListTodosParams) {
	resp := ListTodosResp{
		Items: []Todo{},
		Page:  params.Page,
	}
	var queryParams []domain.ListTodoOptions
	if params.Status != nil {
		queryParams = append(queryParams, domain.WithStatus(domain.TodoStatus(*params.Status)))
	}

	todos, hasMore, err := api.ListTodosUseCase.Query(r.Context(), params.Page, params.Pagesize, queryParams...)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list todos: %v", err), http.StatusInternalServerError)
		return
	}

	for _, t := range todos {
		resp.Items = append(resp.Items, Todo{
			Id:              openapi_types.UUID(t.Id),
			Title:           t.Title,
			CreatedAt:       t.CreatedAt,
			EmailAttempts:   t.EmailAttempts,
			EmailLastError:  t.EmailLastError,
			EmailProviderId: t.EmailProviderId,
			EmailStatus:     EmailStatus(t.EmailStatus),
			Status:          TodoStatus(t.Status),
			DueDate:         openapi_types.Date{Time: t.DueDate},
			UpdatedAt:       t.UpdatedAt,
		})
	}
	if hasMore {
		nextPage := params.Page + 1
		resp.NextPage = &nextPage
	}
	if params.Page > 1 {
		prevPage := params.Page - 1
		resp.PreviousPage = &prevPage
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (api *TodoMailerApp) CreateTodo(w http.ResponseWriter, r *http.Request) {
	var req CreateTodoJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errResp := ErrorResp{}
		errResp.Error.Code = BADREQUEST
		errResp.Error.Message = fmt.Sprintf("invalid request body: %v", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	todo, err := api.CreateTodoUseCase.Execute(r.Context(), req.Title, req.DueDate.Time)
	if err != nil {
		errResp := ErrorResp{}
		errResp.Error.Code = INTERNALERROR
		errResp.Error.Message = fmt.Sprintf("failed to create todo: %v", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	resp := Todo{
		Id:              openapi_types.UUID(todo.Id),
		Title:           todo.Title,
		CreatedAt:       todo.CreatedAt,
		EmailAttempts:   todo.EmailAttempts,
		EmailLastError:  todo.EmailLastError,
		EmailProviderId: todo.EmailProviderId,
		EmailStatus:     EmailStatus(todo.EmailStatus),
		Status:          TodoStatus(todo.Status),
		DueDate:         openapi_types.Date{Time: todo.DueDate},
		UpdatedAt:       todo.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}
func (api *TodoMailerApp) UpdateTodo(w http.ResponseWriter, r *http.Request, todoId openapi_types.UUID) {
	var req UpdateTodoJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errResp := ErrorResp{}
		errResp.Error.Code = BADREQUEST
		errResp.Error.Message = fmt.Sprintf("invalid request body: %v", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	var dueDate *time.Time
	if req.DueDate != nil {
		dueDate = &req.DueDate.Time
	}

	todo, err := api.UpdateTodoUseCase.Execute(
		r.Context(),
		uuid.UUID(todoId),
		req.Title,
		(*domain.TodoStatus)(req.Status),
		dueDate,
	)
	if err != nil {
		errResp := ErrorResp{}
		errResp.Error.Code = INTERNALERROR
		errResp.Error.Message = fmt.Sprintf("failed to update todo: %v", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	resp := Todo{
		Id:              openapi_types.UUID(todo.Id),
		Title:           todo.Title,
		CreatedAt:       todo.CreatedAt,
		EmailAttempts:   todo.EmailAttempts,
		EmailLastError:  todo.EmailLastError,
		EmailProviderId: todo.EmailProviderId,
		EmailStatus:     EmailStatus(todo.EmailStatus),
		Status:          TodoStatus(todo.Status),
		DueDate:         openapi_types.Date{Time: todo.DueDate},
		UpdatedAt:       todo.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

//go:embed webappdist/*
var embedFS embed.FS

// Run starts the HTTP server for the TodoMailerApp.
func (api *TodoMailerApp) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	// Serve webapp static files
	sub, err := fs.Sub(embedFS, "webappdist")
	if err != nil {
		return fmt.Errorf("failed to create sub filesystem for webapp: %w", err)
	}
	mux.Handle("/", http.FileServerFS(sub))

	// get an `http.Handler` that we can use
	h := HandlerWithOptions(api, StdHTTPServerOptions{
		BaseRouter: mux,
		Middlewares: []MiddlewareFunc{
			otelhttp.NewMiddleware(
				"todomailer-api",
				otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
					return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
				}),
			)},
	})

	s := &http.Server{
		Handler: h,
		Addr:    fmt.Sprintf(":%d", api.Port),
	}

	errCh := make(chan error, 1)
	go func() {
		api.Logger.Printf("TodoMailerApp: Listening on port %d", api.Port)
		errCh <- s.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		api.Logger.Print("TodoMailerApp: Shutting down")
		return s.Shutdown(ctx)
	case err := <-errCh:
		return err
	}
}

// IsReady checks if the TodoMailerApp HTTP server is ready by performing a health check.
func (api *TodoMailerApp) IsReady(ctx context.Context) error {
	resp, err := http.Get(fmt.Sprintf("http://:%d", api.Port))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
