package worker

import (
	"context"
	"log"
	"time"
)

type EmailSender struct {
	Logger   *log.Logger   `resolve:""`
	Interval time.Duration `config:"EMAIL_SENDER_INTERVAL" default:"2s"`
}

func (esw *EmailSender) Run(ctx context.Context) error {
	esw.Logger.Println("EmailSender: running...")
	ticker := time.NewTicker(esw.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			esw.Logger.Println("EmailSender: stopping...")
			return nil
		case <-ticker.C:
			esw.Logger.Println("EmailSender: sending emails...")
			// Here would be the logic to send emails
		}
	}
}
