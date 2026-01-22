package email

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/tracing"
	"github.com/google/uuid"
)

// EmailSender is an implementation of domain.EmailSender.
type EmailSender struct {
}

// SendEmail sends an email and returns a transaction ID.
func (es EmailSender) SendEmail(ctx context.Context, to string, subject string, body string) (string, error) {
	_, span := tracing.Start(ctx)
	defer span.End()
	// Here would be the logic to send an email using an external service
	// For this example, we'll just simulate sending an email and return a mock transaction ID
	return uuid.NewString(), nil
}

// NewEmailSender creates a new instance of EmailSender.
func NewEmailSender() EmailSender {
	return EmailSender{}
}

// InitEmailSender initializes the EmailSender and registers it in the dependency container.
type InitEmailSender struct {
}

// Initialize registers the EmailSender in the dependency container.
func (ie InitEmailSender) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.EmailSender](NewEmailSender())
	return ctx, nil
}
