package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	domain_mocks "github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateTodoImpl_Execute(t *testing.T) {
	fixedUUID := func() uuid.UUID {
		return uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	todo := domain.Todo{
		ID:          fixedUUID(),
		Title:       "My new todo",
		Status:      domain.TodoStatus_OPEN,
		EmailStatus: domain.EmailStatus_PENDING,
		CreatedAt:   fixedTime,
		UpdatedAt:   fixedTime,
		DueDate:     fixedTime,
	}

	tests := map[string]struct {
		setExpectations func(repo *domain_mocks.MockTodoRepository, timeService *domain_mocks.MockTimeService, publisher *domain_mocks.MockTodoEventPublisher)
		title           string
		dueDate         time.Time
		expectedTodo    domain.Todo
		expectedErr     error
	}{
		"success": {
			title:   "My new todo",
			dueDate: fixedTime,
			setExpectations: func(repo *domain_mocks.MockTodoRepository, timeService *domain_mocks.MockTimeService, publisher *domain_mocks.MockTodoEventPublisher) {
				timeService.EXPECT().Now().Return(fixedTime)

				repo.EXPECT().CreateTodo(
					mock.Anything,
					todo,
				).Return(nil)

				publisher.EXPECT().PublishEvent(
					mock.Anything,
					domain.TodoEvent{
						Type:   domain.TodoEventType_TODO_CREATED,
						TodoID: fixedUUID(),
					},
				).Return(nil)
			},
			expectedTodo: todo,
			expectedErr:  nil,
		},
		"validation-error-short-title": {
			title:           "Hi",
			dueDate:         fixedTime,
			setExpectations: nil,
			expectedTodo:    domain.Todo{},
			expectedErr:     domain.NewValidationErr("title must be between 3 and 200 characters"),
		},
		"validation-error-long-title": {
			title: func() string {
				longTitle := ""
				for i := 0; i < 201; i++ {
					longTitle += "a"
				}
				return longTitle
			}(),
			dueDate:         fixedTime,
			setExpectations: nil,
			expectedTodo:    domain.Todo{},
			expectedErr:     domain.NewValidationErr("title must be between 3 and 200 characters"),
		},
		"repository-error": {
			title:   "My new todo",
			dueDate: fixedTime,
			setExpectations: func(repo *domain_mocks.MockTodoRepository, timeService *domain_mocks.MockTimeService, publisher *domain_mocks.MockTodoEventPublisher) {
				timeService.EXPECT().Now().Return(fixedTime)

				repo.EXPECT().CreateTodo(
					mock.Anything,
					todo,
				).Return(errors.New("database error"))
			},
			expectedTodo: domain.Todo{},
			expectedErr:  errors.New("database error"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo := domain_mocks.NewMockTodoRepository(t)
			timeService := domain_mocks.NewMockTimeService(t)
			publisher := domain_mocks.NewMockTodoEventPublisher(t)
			if tt.setExpectations != nil {
				tt.setExpectations(repo, timeService, publisher)
			}

			cti := NewCreateTodoImpl(repo, timeService, publisher)
			cti.createUUID = fixedUUID

			got, gotErr := cti.Execute(context.Background(), tt.title, tt.dueDate)
			assert.Equal(t, tt.expectedErr, gotErr)
			assert.Equal(t, tt.expectedTodo, got)
		})
	}
}

func TestInitCreateTodo_Initialize(t *testing.T) {
	ict := InitCreateTodo{}

	ctx, err := ict.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredCreateTodo, err := depend.Resolve[CreateTodo]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredCreateTodo)

}
