package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/usecases"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type TodoMailerAPI struct {
	ServerInterface
	Port              int                 `config:"HTTP_PORT" default:"8080"`
	Logger            *log.Logger         `resolve:""`
	ListTodosUseCase  usecases.ListTodos  `resolve:""`
	CreateTodoUseCase usecases.CreateTodo `resolve:""`
	UpdateTodoUseCase usecases.UpdateTodo `resolve:""`
}

func (api *TodoMailerAPI) ListTodos(w http.ResponseWriter, r *http.Request, params ListTodosParams) {
	// resp := ListTodosResponse{
	// 	Items: []Todo{
	// 		{
	// 			Id:              openapi_types.UUID(uuid.MustParse("11111111-1111-1111-1111-111111111111")),
	// 			Title:           "Sample Todo",
	// 			CreatedAt:       time.Date(2026, 1, 16, 8, 10, 0, 0, time.UTC),
	// 			EmailAttempts:   0,
	// 			EmailLastError:  nil,
	// 			EmailProviderId: nil,
	// 			EmailStatus:     PENDING,
	// 			Status:          OPEN,
	// 			UpdatedAt:       time.Date(2026, 1, 16, 8, 10, 2, 0, time.UTC),
	// 		},
	// 		{
	// 			Id:              openapi_types.UUID(uuid.MustParse("33333333-3333-3333-3333-333333333333")),
	// 			Title:           "Another Todo",
	// 			CreatedAt:       time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
	// 			EmailAttempts:   1,
	// 			EmailLastError:  ptr("SMTP server not reachable"),
	// 			EmailProviderId: ptr("provider-xyz"),
	// 			EmailStatus:     FAILED,
	// 			Status:          DONE,
	// 			UpdatedAt:       time.Date(2026, 1, 15, 12, 5, 0, 0, time.UTC),
	// 		},
	// 		{
	// 			Id:              openapi_types.UUID(uuid.MustParse("44444444-4444-4444-4444-444444444444")),
	// 			Title:           "Completed Todo",
	// 			CreatedAt:       time.Date(2026, 1, 14, 9, 30, 0, 0, time.UTC),
	// 			EmailAttempts:   1,
	// 			EmailLastError:  nil,
	// 			EmailProviderId: ptr("provider-abc"),
	// 			EmailStatus:     SENT,
	// 			Status:          DONE,
	// 			UpdatedAt:       time.Date(2026, 1, 14, 9, 45, 0, 0, time.UTC),
	// 		},
	// 	},
	// 	Page: 1,
	// }
	resp := ListTodosResponse{
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

func (api *TodoMailerAPI) CreateTodo(w http.ResponseWriter, r *http.Request) {
	// resp := Todo{
	// 	Id:              openapi_types.UUID(uuid.MustParse("22222222-2222-2222-2222-222222222222")),
	// 	Title:           "New Todo",
	// 	CreatedAt:       time.Date(2026, 1, 16, 9, 0, 0, 0, time.UTC),
	// 	EmailAttempts:   0,
	// 	EmailLastError:  nil,
	// 	EmailProviderId: nil,
	// 	EmailStatus:     PENDING,
	// 	Status:          OPEN,
	// 	UpdatedAt:       time.Date(2026, 1, 16, 9, 0, 0, 0, time.UTC),
	// }

	var req CreateTodoJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errResp := ErrorResponse{}
		errResp.Error.Code = BADREQUEST
		errResp.Error.Message = fmt.Sprintf("invalid request body: %v", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	todo, err := api.CreateTodoUseCase.Execute(r.Context(), req.Title)
	if err != nil {
		errResp := ErrorResponse{}
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
		UpdatedAt:       todo.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}
func (api *TodoMailerAPI) UpdateTodo(w http.ResponseWriter, r *http.Request, todoId openapi_types.UUID) {
	// resp := Todo{
	// 	Id:              openapi_types.UUID(uuid.MustParse("22222222-2222-2222-2222-222222222222")),
	// 	Title:           "New Todo",
	// 	CreatedAt:       time.Date(2026, 1, 16, 9, 0, 0, 0, time.UTC),
	// 	EmailAttempts:   0,
	// 	EmailLastError:  nil,
	// 	EmailProviderId: nil,
	// 	EmailStatus:     PENDING,
	// 	Status:          OPEN,
	// 	UpdatedAt:       time.Date(2026, 1, 16, 9, 0, 0, 0, time.UTC),
	// }

	var req UpdateTodoJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errResp := ErrorResponse{}
		errResp.Error.Code = BADREQUEST
		errResp.Error.Message = fmt.Sprintf("invalid request body: %v", err)

		_ = json.NewEncoder(w).Encode(errResp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	todo, err := api.UpdateTodoUseCase.Execute(r.Context(), uuid.UUID(todoId), req.Title, (*domain.TodoStatus)(req.Status))
	if err != nil {
		errResp := ErrorResponse{}
		errResp.Error.Code = INTERNALERROR
		errResp.Error.Message = fmt.Sprintf("failed to update todo: %v", err)

		_ = json.NewEncoder(w).Encode(errResp)
		w.WriteHeader(http.StatusInternalServerError)
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
		UpdatedAt:       todo.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (api *TodoMailerAPI) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	// Serve webapp static files
	fs := http.FileServer(http.Dir("./webapp/dist"))
	mux.Handle("/", fs)

	// get an `http.Handler` that we can use
	// Custom span name formatter
	spanNameFormatter := func(operation string, r *http.Request) string {
		return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
	}

	h := HandlerWithOptions(api, StdHTTPServerOptions{
		BaseRouter: mux,
		Middlewares: []MiddlewareFunc{
			otelhttp.NewMiddleware(
				"todomailer-api",
				otelhttp.WithSpanNameFormatter(spanNameFormatter),
			)},
	})

	s := &http.Server{
		Handler: h,
		Addr:    fmt.Sprintf(":%d", api.Port),
	}

	errCh := make(chan error, 1)
	go func() {
		api.Logger.Printf("TodoMailerAPIServer: Listening on port %d", api.Port)
		errCh <- s.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		api.Logger.Print("TodoMailerAPIServer: Shutting down")
		return s.Shutdown(ctx)
	case err := <-errCh:
		return err
	}
}
