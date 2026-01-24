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

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/openapi"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// TodoAppServer is the HTTP server adapter for the TodoApp application.
//
// It implements the OpenAPI-generated ServerInterface and serves both the REST API
// endpoints and the embedded web application static files. The server is instrumented
// with OpenTelemetry for distributed tracing and configured via environment variables
// or configuration providers through the symbiont framework.
//
// Dependencies are automatically resolved and injected at initialization time.
type TodoAppServer struct {
	Port                   int                      `config:"HTTP_PORT" default:"8080"`
	Logger                 *log.Logger              `resolve:""`
	ListTodosUseCase       usecases.ListTodos       `resolve:""`
	CreateTodoUseCase      usecases.CreateTodo      `resolve:""`
	UpdateTodoUseCase      usecases.UpdateTodo      `resolve:""`
	DeleteTodoUseCase      usecases.DeleteTodo      `resolve:""`
	GetBoardSummaryUseCase usecases.GetBoardSummary `resolve:""`
}

func (api TodoAppServer) ListTodos(w http.ResponseWriter, r *http.Request, params openapi.ListTodosParams) {
	resp := openapi.ListTodosResp{
		Items: []openapi.Todo{},
		Page:  params.Page,
	}
	var queryParams []domain.ListTodoOptions
	if params.Status != nil {
		queryParams = append(queryParams, domain.WithStatus(domain.TodoStatus(*params.Status)))
	}

	todos, hasMore, err := api.ListTodosUseCase.Query(r.Context(), params.Page, params.Pagesize, queryParams...)
	if err != nil {
		respondError(w, toOpenAPIError(err))
		return
	}

	for _, t := range todos {
		resp.Items = append(resp.Items, toOpenAPITodo(t))
	}
	if hasMore {
		nextPage := params.Page + 1
		resp.NextPage = &nextPage
	}
	if params.Page > 1 {
		prevPage := params.Page - 1
		resp.PreviousPage = &prevPage
	}

	respondJSON(w, http.StatusOK, resp)
}

func (api TodoAppServer) CreateTodo(w http.ResponseWriter, r *http.Request) {
	var req openapi.CreateTodoJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errResp := openapi.ErrorResp{}
		errResp.Error.Code = openapi.BADREQUEST
		errResp.Error.Message = fmt.Sprintf("invalid request body: %v", err)

		respondError(w, errResp)
		return
	}

	todo, err := api.CreateTodoUseCase.Execute(r.Context(), req.Title, req.DueDate.Time)
	if err != nil {
		respondError(w, toOpenAPIError(err))
		return
	}

	respondJSON(w, http.StatusCreated, toOpenAPITodo(todo))
}
func (api TodoAppServer) UpdateTodo(w http.ResponseWriter, r *http.Request, todoId openapi_types.UUID) {
	var req openapi.UpdateTodoJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errResp := openapi.ErrorResp{}
		errResp.Error.Code = openapi.BADREQUEST
		errResp.Error.Message = fmt.Sprintf("invalid request body: %v", err)

		respondError(w, errResp)
		return
	}

	var dueDate *time.Time
	if req.DueDate != nil {
		dueDate = &req.DueDate.Time
	}
	if req.Status != nil && *req.Status != openapi.DONE && *req.Status != openapi.OPEN {
		errResp := openapi.ErrorResp{}
		errResp.Error.Code = openapi.BADREQUEST
		errResp.Error.Message = fmt.Sprintf("invalid request body: unknown TodoStatus value: %s", *req.Status)

		respondError(w, errResp)
		return
	}

	todo, err := api.UpdateTodoUseCase.Execute(
		r.Context(),
		uuid.UUID(todoId),
		req.Title,
		(*domain.TodoStatus)(req.Status),
		dueDate,
	)
	if err != nil {
		respondError(w, toOpenAPIError(err))
		return
	}

	respondJSON(w, http.StatusOK, toOpenAPITodo(todo))
}

func (api TodoAppServer) DeleteTodo(w http.ResponseWriter, r *http.Request, todoId openapi_types.UUID) {
	err := api.DeleteTodoUseCase.Execute(r.Context(), uuid.UUID(todoId))
	if err != nil {
		respondError(w, toOpenAPIError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api TodoAppServer) GetBoardSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := api.GetBoardSummaryUseCase.Query(r.Context())
	if err != nil {
		respondError(w, toOpenAPIError(err))
		return
	}

	resp := openapi.BoardSummary{
		Counts: openapi.TodoStatusCounts{
			DONE: summary.Content.Counts.Done,
			OPEN: summary.Content.Counts.Open,
		},
		NearDeadline: summary.Content.NearDeadline,
		NextUp:       []openapi.NextUpTodoItem{},
		Overdue:      summary.Content.Overdue,
		Summary:      summary.Content.Summary,
	}
	for _, item := range summary.Content.NextUp {
		resp.NextUp = append(resp.NextUp, openapi.NextUpTodoItem{
			Title:  item.Title,
			Reason: item.Reason,
		})
	}

	respondJSON(w, http.StatusOK, resp)
}

func (api TodoAppServer) ClearChatMessages(w http.ResponseWriter, r *http.Request) {

}

func (api TodoAppServer) ListChatMessages(w http.ResponseWriter, r *http.Request, params openapi.ListChatMessagesParams) {
}

func (api TodoAppServer) StreamChat(w http.ResponseWriter, r *http.Request) {

}

//go:embed webappdist/*
var embedFS embed.FS

// Run starts the HTTP server for the TodoAppServer.
func (api TodoAppServer) Run(ctx context.Context) error {

	mux := http.NewServeMux()

	// Serve webapp static files
	sub, err := fs.Sub(embedFS, "webappdist")
	if err != nil {
		return fmt.Errorf("failed to create sub filesystem for webapp: %w", err)
	}
	mux.Handle("/", http.FileServerFS(sub))

	// get an `http.Handler` that we can use
	h := openapi.HandlerWithOptions(api, openapi.StdHTTPServerOptions{
		BaseRouter: mux,
		Middlewares: []openapi.MiddlewareFunc{
			otelhttp.NewMiddleware(
				"todoapp-api",
				otelhttp.WithSpanNameFormatter(tracing.SpanNameFormatter),
			)},
	})

	s := &http.Server{
		Handler: h,
		Addr:    fmt.Sprintf(":%d", api.Port),
	}

	errCh := make(chan error, 1)
	go func() {
		api.Logger.Printf("TodoAppServer: Listening on port %d", api.Port)
		errCh <- s.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		api.Logger.Print("TodoAppServer: Shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// IsReady checks if the TodoAppServer is ready by performing a health check.
func (api TodoAppServer) IsReady(ctx context.Context) error {
	resp, err := http.Get(fmt.Sprintf("http://:%d", api.Port))
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func respondJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, err openapi.ErrorResp) {
	statusCode := http.StatusInternalServerError
	switch err.Error.Code {
	case openapi.BADREQUEST:
		statusCode = http.StatusBadRequest
	case openapi.NOTFOUND:
		statusCode = http.StatusNotFound
	}
	respondJSON(w, statusCode, err)
}

var _ openapi.ServerInterface = (*TodoAppServer)(nil)
