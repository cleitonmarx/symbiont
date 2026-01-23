package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/openapi"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases/mocks"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	dueDate    = time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC)
	domainTodo = domain.Todo{
		ID:        uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Title:     "Buy groceries",
		Status:    domain.TodoStatus_DONE,
		DueDate:   dueDate,
		CreatedAt: time.Date(2026, 1, 22, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 22, 10, 30, 0, 0, time.UTC),
	}
	restTodo = openapi.Todo{
		Id:        openapi_types.UUID(domainTodo.ID),
		Title:     domainTodo.Title,
		Status:    openapi.DONE,
		DueDate:   openapi_types.Date{Time: domainTodo.DueDate},
		CreatedAt: domainTodo.CreatedAt,
		UpdatedAt: domainTodo.UpdatedAt,
	}
)

func TestTodoAppServer_CreateTodo(t *testing.T) {
	tests := map[string]struct {
		requestBody    []byte
		setupMocks     func(*mocks.MockCreateTodo)
		expectedStatus int
		expectedBody   *openapi.Todo
		expectedError  *openapi.ErrorResp
	}{
		"success": {
			requestBody: serializeJSON(t, openapi.CreateTodoJSONRequestBody{
				Title:   "Buy groceries",
				DueDate: openapi_types.Date{Time: dueDate},
			}),
			setupMocks: func(m *mocks.MockCreateTodo) {
				m.EXPECT().
					Execute(mock.Anything, "Buy groceries", dueDate).Return(domainTodo, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   &restTodo,
		},
		"bad-request": {
			requestBody: serializeJSON(t, openapi.CreateTodoJSONRequestBody{
				DueDate: openapi_types.Date{Time: dueDate},
			}),
			setupMocks: func(m *mocks.MockCreateTodo) {
				m.EXPECT().
					Execute(mock.Anything, "", dueDate).
					Return(domain.Todo{}, domain.NewValidationErr("title is required"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.BADREQUEST,
					Message: "title is required",
				},
			},
		},
		"invalid-json-body": {
			requestBody:    []byte(`{"title": "Test todo", "due_date": "invalid-date"}`),
			setupMocks:     func(m *mocks.MockCreateTodo) {},
			expectedStatus: http.StatusBadRequest,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.BADREQUEST,
					Message: "invalid request body: parsing time \"invalid-date\" as \"2006-01-02\": cannot parse \"invalid-date\" as \"2006\"",
				},
			},
		},
		"internal-server-error": {
			requestBody: serializeJSON(t, openapi.CreateTodoJSONRequestBody{
				Title:   "Test todo",
				DueDate: openapi_types.Date{Time: time.Time{}},
			}),
			setupMocks: func(m *mocks.MockCreateTodo) {
				m.EXPECT().
					Execute(mock.Anything, "Test todo", time.Time{}).
					Return(domain.Todo{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.INTERNALERROR,
					Message: "internal server error",
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockCreateTodo := mocks.NewMockCreateTodo(t)
			if tt.setupMocks != nil {
				tt.setupMocks(mockCreateTodo)
			}

			server := &TodoAppServer{
				CreateTodoUseCase: mockCreateTodo,
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			openapi.Handler(server).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response openapi.Todo
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody.Id, response.Id)
				assert.Equal(t, tt.expectedBody.Title, response.Title)
				assert.Equal(t, tt.expectedBody.Status, response.Status)
			}

			if tt.expectedError != nil {
				var response openapi.ErrorResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError.Error, response.Error)
			}

			mockCreateTodo.AssertExpectations(t)
		})
	}
}

func TestTodoAppServer_ListTodos(t *testing.T) {
	tests := map[string]struct {
		page           int
		pageSize       int
		todoStatus     *openapi.TodoStatus
		setupMocks     func(*mocks.MockListTodos)
		expectedStatus int
		expectedBody   *openapi.ListTodosResp
		expectedError  *openapi.ErrorResp
	}{
		"success-with-todos": {
			page:     1,
			pageSize: 1,
			setupMocks: func(m *mocks.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 1, mock.Anything).
					Return([]domain.Todo{domainTodo}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &openapi.ListTodosResp{
				Items: []openapi.Todo{toOpenAPITodo(domainTodo)},
				Page:  1,
			},
		},
		"success-with-no-todos": {
			page:     1,
			pageSize: 1,
			setupMocks: func(m *mocks.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 1, mock.Anything).
					Return([]domain.Todo{}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &openapi.ListTodosResp{
				Items: []openapi.Todo{},
				Page:  1,
			},
		},
		"success-with-next-and-previous-page": {
			page:     2,
			pageSize: 1,
			setupMocks: func(m *mocks.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 2, 1, mock.Anything).
					Return([]domain.Todo{domainTodo}, true, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &openapi.ListTodosResp{
				Items:        []openapi.Todo{toOpenAPITodo(domainTodo)},
				Page:         2,
				NextPage:     common.Ptr(3),
				PreviousPage: common.Ptr(1),
			},
		},
		"success-with-status-filter": {
			page:     1,
			pageSize: 10,
			todoStatus: func() *openapi.TodoStatus {
				s := openapi.DONE
				return &s
			}(),
			setupMocks: func(m *mocks.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 10, mock.Anything).
					Run(func(_ context.Context, _ int, _ int, opts ...domain.ListTodoOptions) {
						p := domain.ListTodosParams{}
						for _, opt := range opts {
							opt(&p)
						}
						assert.Equal(t, domain.TodoStatus_DONE, *p.Status)
					}).
					Return([]domain.Todo{domainTodo}, false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &openapi.ListTodosResp{
				Items: []openapi.Todo{restTodo},
				Page:  1,
			},
		},
		"use-case-error": {
			page:     1,
			pageSize: 10,
			setupMocks: func(m *mocks.MockListTodos) {
				m.EXPECT().
					Query(mock.Anything, 1, 10, mock.Anything).
					Return(nil, false, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.INTERNALERROR,
					Message: "internal server error",
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockListTodos := mocks.NewMockListTodos(t)
			tt.setupMocks(mockListTodos)

			server := &TodoAppServer{
				ListTodosUseCase: mockListTodos,
			}

			u, err := url.Parse("http://localhost/api/v1/todos")
			assert.NoError(t, err)
			q := u.Query()
			q.Set("page", strconv.Itoa(tt.page))
			q.Set("pagesize", strconv.Itoa(tt.pageSize))
			if tt.todoStatus != nil {
				q.Set("status", string(*tt.todoStatus))
			}
			u.RawQuery = q.Encode()
			req := httptest.NewRequest(http.MethodGet, u.String(), nil)

			w := httptest.NewRecorder()

			openapi.Handler(server).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response openapi.ListTodosResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedBody, response)
			}

			if tt.expectedError != nil {
				var response openapi.ErrorResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedError, response)
			}

			mockListTodos.AssertExpectations(t)
		})
	}
}

func TestTodoAppServer_UpdateTodo(t *testing.T) {
	tests := map[string]struct {
		todoID         string
		requestBody    []byte
		setupMocks     func(*mocks.MockUpdateTodo)
		expectedStatus int
		expectedBody   *openapi.Todo
		expectedError  *openapi.ErrorResp
	}{
		"success": {
			todoID: domainTodo.ID.String(),
			requestBody: serializeJSON(t, openapi.UpdateTodoJSONRequestBody{
				Title:   common.Ptr("Buy groceries"),
				Status:  common.Ptr(openapi.DONE),
				DueDate: &openapi_types.Date{Time: dueDate},
			}),
			setupMocks: func(m *mocks.MockUpdateTodo) {
				m.EXPECT().
					Execute(mock.Anything, domainTodo.ID, common.Ptr("Buy groceries"), common.Ptr(domain.TodoStatus_DONE), &dueDate).
					Return(domainTodo, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   &restTodo,
		},
		"todo-not-found": {
			todoID: domainTodo.ID.String(),
			requestBody: serializeJSON(t, openapi.UpdateTodoJSONRequestBody{
				Status: common.Ptr(openapi.DONE),
			}),
			setupMocks: func(m *mocks.MockUpdateTodo) {
				m.EXPECT().
					Execute(mock.Anything, domainTodo.ID, (*string)(nil), common.Ptr(domain.TodoStatus_DONE), (*time.Time)(nil)).
					Return(domain.Todo{}, domain.NewNotFoundErr("todo not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.NOTFOUND,
					Message: "todo not found",
				},
			},
		},
		"invalid-status": {
			todoID:         domainTodo.ID.String(),
			requestBody:    []byte(`{"status": "INVALID_STATUS"}`),
			setupMocks:     func(m *mocks.MockUpdateTodo) {},
			expectedStatus: http.StatusBadRequest,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.BADREQUEST,
					Message: "invalid request body: unknown TodoStatus value: INVALID_STATUS",
				},
			},
		},
		"invalid-json-body": {
			todoID:         domainTodo.ID.String(),
			requestBody:    []byte(`{"title": "Test todo", "due_date": "invalid-date"}`),
			setupMocks:     func(m *mocks.MockUpdateTodo) {},
			expectedStatus: http.StatusBadRequest,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.BADREQUEST,
					Message: "invalid request body: error reading 'due_date': parsing time \"invalid-date\" as \"2006-01-02\": cannot parse \"invalid-date\" as \"2006\"",
				},
			},
		},
		"use-case-error": {
			todoID: domainTodo.ID.String(),
			requestBody: serializeJSON(t, openapi.UpdateTodoJSONRequestBody{
				Status: common.Ptr(openapi.DONE),
			}),
			setupMocks: func(m *mocks.MockUpdateTodo) {
				m.EXPECT().
					Execute(mock.Anything, domainTodo.ID, (*string)(nil), common.Ptr(domain.TodoStatus_DONE), (*time.Time)(nil)).
					Return(domain.Todo{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.INTERNALERROR,
					Message: "internal server error",
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockUpdateTodo := mocks.NewMockUpdateTodo(t)
			tt.setupMocks(mockUpdateTodo)
			server := &TodoAppServer{
				UpdateTodoUseCase: mockUpdateTodo,
			}

			req := httptest.NewRequest(http.MethodPatch, "/api/v1/todos/"+tt.todoID, bytes.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			openapi.Handler(server).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				var response openapi.Todo
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedBody, response)
			}
			if tt.expectedError != nil {
				var response openapi.ErrorResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedError, response)
			}
		})
	}

}

func TestTodoAppServer_GetBoardSummary(t *testing.T) {
	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	generatedAt := time.Date(2026, 1, 22, 10, 30, 0, 0, time.UTC)

	tests := map[string]struct {
		setupMocks     func(*mocks.MockGetBoardSummary)
		expectedStatus int
		expectedBody   *openapi.BoardSummary
		expectedError  *openapi.ErrorResp
	}{
		"success": {
			setupMocks: func(m *mocks.MockGetBoardSummary) {
				m.EXPECT().Query(mock.Anything).Return(domain.BoardSummary{
					ID:            fixedUUID,
					Model:         "ai/gpt-oss:latest",
					GeneratedAt:   generatedAt,
					SourceVersion: 1,
					Content: domain.BoardSummaryContent{
						Counts: domain.TodoStatusCounts{
							Open: 5,
							Done: 3,
						},
						NextUp: []domain.NextUpTodoItem{
							{
								Title:  "Buy groceries",
								Reason: "Due tomorrow",
							},
						},
						Overdue:      []string{"Pay electricity bill"},
						NearDeadline: []string{"Renew car insurance"},
						Summary:      "You have 1 overdue task and 2 tasks due this week.",
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &openapi.BoardSummary{
				Counts: openapi.TodoStatusCounts{
					OPEN: 5,
					DONE: 3,
				},
				NextUp: []openapi.NextUpTodoItem{
					{
						Title:  "Buy groceries",
						Reason: "Due tomorrow",
					},
				},
				Overdue:      []string{"Pay electricity bill"},
				NearDeadline: []string{"Renew car insurance"},
				Summary:      "You have 1 overdue task and 2 tasks due this week.",
			},
		},
		"summary-not-found": {
			setupMocks: func(m *mocks.MockGetBoardSummary) {
				m.EXPECT().
					Query(mock.Anything).
					Return(domain.BoardSummary{}, domain.NewNotFoundErr("board summary not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.NOTFOUND,
					Message: "board summary not found",
				},
			},
		},
		"use-case-error": {
			setupMocks: func(m *mocks.MockGetBoardSummary) {
				m.EXPECT().
					Query(mock.Anything).
					Return(domain.BoardSummary{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError: &openapi.ErrorResp{
				Error: openapi.Error{
					Code:    openapi.INTERNALERROR,
					Message: "internal server error",
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockGetBoardSummary := mocks.NewMockGetBoardSummary(t)
			tt.setupMocks(mockGetBoardSummary)

			server := &TodoAppServer{
				GetBoardSummaryUseCase: mockGetBoardSummary,
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/board/summary", nil)
			w := httptest.NewRecorder()

			server.GetBoardSummary(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response openapi.BoardSummary
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedBody, response)
			}

			if tt.expectedError != nil {
				var response openapi.ErrorResp
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError.Error, response.Error)
			}

			mockGetBoardSummary.AssertExpectations(t)
		})
	}
}

func TestTodoAppServer_Run(t *testing.T) {
	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &TodoAppServer{
		Port:   12345,
		Logger: log.Default(),
	}

	shutdownCh := make(chan error, 1)

	go func() {
		shutdownCh <- server.Run(cancelCtx)
	}()

	for range 10 {
		err := server.IsReady(cancelCtx)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-shutdownCh:
		if err != nil {
			assert.Fail(t, "server exited with error: %v", err)
		}
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		assert.Fail(t, "server did not shut down in time")

	}
}

// serializeJSON is a helper function to marshal a value to JSON for test requests.
func serializeJSON(t *testing.T, v any) []byte {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return data
}
