package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/tracing"
)

// SendDoneTodoEmails is the use case interface for sending emails for done todos.
type SendDoneTodoEmails interface {
	Execute(ctx context.Context) error
}

// SendDoneTodoEmailsImpl is the implementation of SendDoneTodoEmails use case.
type SendDoneTodoEmailsImpl struct {
	repo   domain.Repository
	sender domain.EmailSender
	time   domain.TimeService
}

// NewSendDoneTodoEmailsImpl creates a new instance of SendDoneTodoEmailsImpl.
func NewSendDoneTodoEmailsImpl(repo domain.Repository, sender domain.EmailSender, time domain.TimeService) SendDoneTodoEmailsImpl {
	return SendDoneTodoEmailsImpl{
		repo:   repo,
		sender: sender,
		time:   time,
	}
}

// Execute sends emails for all done todos that have not been emailed yet.
func (se SendDoneTodoEmailsImpl) Execute(ctx context.Context) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	todos, _, err := se.repo.ListTodos(
		spanCtx,
		1,
		100,
		domain.WithEmailStatuses(domain.EmailStatus_PENDING, domain.EmailStatus_FAILED),
		domain.WithStatus(domain.TodoStatus_DONE),
	)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	for _, todo := range todos {
		transID, err := se.sender.SendEmail(spanCtx, "admin", "Todo Completed: "+todo.Title, "The todo item has been completed.")
		if tracing.RecordErrorAndStatus(span, err) {
			todo.EmailStatus = domain.EmailStatus_FAILED
			todo.EmailAttempts += 1
		} else {
			todo.EmailStatus = domain.EmailStatus_SENT
			todo.EmailProviderId = &transID
		}
		todo.UpdatedAt = se.time.Now()
		err = se.repo.UpdateTodo(spanCtx, todo)
		if tracing.RecordErrorAndStatus(span, err) {
			return err
		}
	}

	return nil
}

// InitSendDoneTodoEmails is the initializer for SendDoneTodoEmails use case.
type InitSendDoneTodoEmails struct {
	Repo   domain.Repository  `resolve:""`
	Sender domain.EmailSender `resolve:""`
	Time   domain.TimeService `resolve:""`
}

// Initialize registers the SendDoneTodoEmails implementation in the dependency container.
func (ie *InitSendDoneTodoEmails) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[SendDoneTodoEmails](NewSendDoneTodoEmailsImpl(ie.Repo, ie.Sender, ie.Time))
	return ctx, nil
}
