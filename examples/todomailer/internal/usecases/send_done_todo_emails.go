package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/tracing"
)

// CompletedTodoEmailQueue is a channel type for sending processed domain.Todo items.
type CompletedTodoEmailQueue chan domain.Todo

// SendDoneTodoEmails is the use case interface for sending emails for done todos.
type SendDoneTodoEmails interface {
	Execute(ctx context.Context) error
}

// SendDoneTodoEmailsImpl is the implementation of SendDoneTodoEmails use case.
type SendDoneTodoEmailsImpl struct {
	todoRepo domain.TodoRepository
	sender   domain.EmailSender
	time     domain.TimeService
	queue    CompletedTodoEmailQueue
}

// NewSendDoneTodoEmailsImpl creates a new instance of SendDoneTodoEmailsImpl.
func NewSendDoneTodoEmailsImpl(todoRepo domain.TodoRepository, sender domain.EmailSender, time domain.TimeService, queue CompletedTodoEmailQueue) SendDoneTodoEmailsImpl {
	return SendDoneTodoEmailsImpl{
		todoRepo: todoRepo,
		sender:   sender,
		time:     time,
		queue:    queue,
	}
}

// Execute sends emails for all done todos that have not been emailed yet.
func (se SendDoneTodoEmailsImpl) Execute(ctx context.Context) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	todos, _, err := se.todoRepo.ListTodos(
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
			todo.EmailProviderID = &transID
		}
		todo.UpdatedAt = se.time.Now()
		err = se.todoRepo.UpdateTodo(spanCtx, todo)
		if tracing.RecordErrorAndStatus(span, err) {
			return err
		}

		if se.queue != nil {
			se.queue <- todo
		}
	}

	return nil
}

// InitSendDoneTodoEmails is the initializer for SendDoneTodoEmails use case.
type InitSendDoneTodoEmails struct {
	TodoRepo domain.TodoRepository `resolve:""`
	Sender   domain.EmailSender    `resolve:""`
	Time     domain.TimeService    `resolve:""`
}

// Initialize registers the SendDoneTodoEmails implementation in the dependency container.
func (ie InitSendDoneTodoEmails) Initialize(ctx context.Context) (context.Context, error) {
	queue, _ := depend.Resolve[CompletedTodoEmailQueue]()
	depend.Register[SendDoneTodoEmails](NewSendDoneTodoEmailsImpl(ie.TodoRepo, ie.Sender, ie.Time, queue))

	return ctx, nil
}
