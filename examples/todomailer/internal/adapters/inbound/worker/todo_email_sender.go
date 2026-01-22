package worker

import (
	"context"
	"log"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/usecases"
)

// TodoEmailSender is a worker that periodically sends emails for completed todo items.
type TodoEmailSender struct {
	Logger     *log.Logger                 `resolve:""`
	Interval   time.Duration               `config:"EMAIL_SENDER_INTERVAL" default:"2s"`
	SendEmails usecases.SendDoneTodoEmails `resolve:""`
}

// Run starts the email sending worker.
func (esw TodoEmailSender) Run(ctx context.Context) error {
	esw.Logger.Println("TodoEmailSender: running...")
	ticker := time.NewTicker(esw.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			esw.Logger.Println("TodoEmailSender: stopping...")
			return nil
		case <-ticker.C:
			err := esw.SendEmails.Execute(ctx)
			if err != nil {
				esw.Logger.Printf("TodoEmailSender: error sending emails: %v", err)
			}
		}
	}
}
