package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUpdateTodoImpl_Execute(t *testing.T) {
	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	todo := domain.Todo{
		Id:      fixedUUID,
		Title:   "Updated Todo",
		Status:  domain.TodoStatus_OPEN,
		DueDate: fixedTime,
	}

	tests := map[string]struct {
		setExpectations func(repo *domain.MockRepository, timeService *domain.MockTimeService)
		id              uuid.UUID
		title           *string
		status          *domain.TodoStatus
		dueDate         *time.Time
		expectedTodo    domain.Todo
		expectedErr     error
	}{
		"success": {
			id:     fixedUUID,
			title:  &todo.Title,
			status: &todo.Status,
			setExpectations: func(repo *domain.MockRepository, timeService *domain.MockTimeService) {
				timeService.EXPECT().Now().Return(fixedTime)
				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(todo, nil)
				repo.EXPECT().UpdateTodo(mock.Anything, mock.MatchedBy(func(t domain.Todo) bool {
					return t.Id == fixedUUID && t.Title == todo.Title && t.UpdatedAt == fixedTime
				})).Return(nil)
			},
			expectedTodo: todo,
			expectedErr:  nil,
		},
		"todo-not-found": {
			id: fixedUUID,
			setExpectations: func(repo *domain.MockRepository, timeService *domain.MockTimeService) {
				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(domain.Todo{}, errors.New("not found"))
			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("not found"),
		},
		"repository-error": {
			id: fixedUUID,
			setExpectations: func(repo *domain.MockRepository, timeService *domain.MockTimeService) {
				timeService.EXPECT().Now().Return(fixedTime)
				repo.EXPECT().GetTodo(mock.Anything, fixedUUID).Return(todo, nil)
				repo.EXPECT().UpdateTodo(mock.Anything, mock.Anything).Return(errors.New("database error"))
			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("database error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo := domain.NewMockRepository(t)
			timeService := domain.NewMockTimeService(t)
			if tt.setExpectations != nil {
				tt.setExpectations(repo, timeService)
			}

			uti := NewUpdateTodoImpl(repo, timeService)

			got, gotErr := uti.Execute(context.Background(), tt.id, tt.title, tt.status, tt.dueDate)
			assert.Equal(t, tt.expectedErr, gotErr)
			if tt.expectedErr == nil {
				assert.Equal(t, tt.id, got.Id)
			}
		})
	}
}

func TestInitUpdateTodo_Initialize(t *testing.T) {
	repo := domain.NewMockRepository(t)
	timeService := domain.NewMockTimeService(t)

	iut := InitUpdateTodo{
		Repo:        repo,
		TimeService: timeService,
	}

	ctx, err := iut.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredUpdateTodo, err := depend.Resolve[UpdateTodo]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredUpdateTodo)
}
