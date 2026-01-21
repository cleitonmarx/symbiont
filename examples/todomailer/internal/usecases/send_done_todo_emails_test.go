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

func TestSendDoneTodoEmailsImpl_Execute(t *testing.T) {
	tests := map[string]struct {
		setExpectations func(repo *domain.MockRepository, sender *domain.MockEmailSender, timeService *domain.MockTimeService)
		expectedErr     error
	}{
		"success": {
			setExpectations: func(repo *domain.MockRepository, sender *domain.MockEmailSender, timeService *domain.MockTimeService) {
				repo.EXPECT().
					ListTodos(mock.Anything, 1, 100, mock.Anything, mock.Anything).
					Return(
						[]domain.Todo{
							{Id: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), Title: "Todo 1", Status: domain.TodoStatus_DONE, EmailStatus: domain.EmailStatus_PENDING},
						},
						false,
						nil,
					)

				trasnUUID := uuid.Nil.String()

				sender.EXPECT().
					SendEmail(mock.Anything, "admin", "Todo Completed: Todo 1", "The todo item has been completed.").
					Return(trasnUUID, nil)

				fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

				timeService.EXPECT().Now().
					Return(fixedTime)

				repo.EXPECT().
					UpdateTodo(mock.Anything, domain.Todo{
						Id:              uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						Title:           "Todo 1",
						Status:          domain.TodoStatus_DONE,
						EmailStatus:     domain.EmailStatus_SENT,
						EmailProviderId: &trasnUUID,
						UpdatedAt:       fixedTime,
					}).
					Return(nil)
			},
			expectedErr: nil,
		},
		"repository-error": {
			setExpectations: func(repo *domain.MockRepository, sender *domain.MockEmailSender, timeService *domain.MockTimeService) {
				repo.EXPECT().ListTodos(mock.Anything, 1, 100, mock.Anything, mock.Anything).Return(nil, false, errors.New("database error"))
			},
			expectedErr: errors.New("database error"),
		},
		"email-sending-error": {
			setExpectations: func(repo *domain.MockRepository, sender *domain.MockEmailSender, timeService *domain.MockTimeService) {
				repo.EXPECT().ListTodos(mock.Anything, 1, 100, mock.Anything, mock.Anything).Return([]domain.Todo{
					{Id: uuid.New(), Title: "Todo 1", Status: domain.TodoStatus_DONE, EmailStatus: domain.EmailStatus_PENDING},
				}, false, nil)

				sender.EXPECT().SendEmail(mock.Anything, "admin", "Todo Completed: Todo 1", "The todo item has been completed.").Return(uuid.Nil.String(), errors.New("email error"))

				fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

				timeService.EXPECT().Now().
					Return(fixedTime)
				repo.EXPECT().UpdateTodo(mock.Anything, mock.MatchedBy(func(t domain.Todo) bool {
					return t.Title == "Todo 1" &&
						t.EmailStatus == domain.EmailStatus_FAILED &&
						t.EmailAttempts == 1 &&
						t.UpdatedAt == fixedTime
				})).Return(nil)
			},
		},
		"update-todo-error": {
			setExpectations: func(repo *domain.MockRepository, sender *domain.MockEmailSender, timeService *domain.MockTimeService) {
				repo.EXPECT().ListTodos(mock.Anything, 1, 100, mock.Anything, mock.Anything).Return([]domain.Todo{
					{Id: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), Title: "Todo 1", Status: domain.TodoStatus_DONE, EmailStatus: domain.EmailStatus_PENDING},
				}, false, nil)

				trasnUUID := uuid.Nil.String()

				sender.EXPECT().
					SendEmail(mock.Anything, "admin", "Todo Completed: Todo 1", "The todo item has been completed.").
					Return(trasnUUID, nil)

				fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

				timeService.EXPECT().Now().
					Return(fixedTime)

				repo.EXPECT().
					UpdateTodo(mock.Anything, domain.Todo{
						Id:              uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						Title:           "Todo 1",
						Status:          domain.TodoStatus_DONE,
						EmailStatus:     domain.EmailStatus_SENT,
						EmailProviderId: &trasnUUID,
						UpdatedAt:       fixedTime,
					}).
					Return(errors.New("database error"))
			},
			expectedErr: errors.New("database error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo := domain.NewMockRepository(t)
			sender := domain.NewMockEmailSender(t)
			timeService := domain.NewMockTimeService(t)
			if tt.setExpectations != nil {
				tt.setExpectations(repo, sender, timeService)
			}

			sdte := NewSendDoneTodoEmailsImpl(repo, sender, timeService, nil)

			gotErr := sdte.Execute(context.Background())
			assert.Equal(t, tt.expectedErr, gotErr)
		})
	}
}

func TestInitSendDoneTodoEmails_Initialize(t *testing.T) {
	repo := domain.NewMockRepository(t)
	sender := domain.NewMockEmailSender(t)

	expectedSendDoneTodoEmails := NewSendDoneTodoEmailsImpl(repo, sender, domain.NewMockTimeService(t), nil)

	ie := InitSendDoneTodoEmails{
		Repo:   repo,
		Sender: sender,
		Time:   domain.NewMockTimeService(t),
	}

	ctx, err := ie.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredSendDoneTodoEmails, err := depend.Resolve[SendDoneTodoEmails]()
	assert.NoError(t, err)
	assert.EqualExportedValues(t, expectedSendDoneTodoEmails, registeredSendDoneTodoEmails)
}
