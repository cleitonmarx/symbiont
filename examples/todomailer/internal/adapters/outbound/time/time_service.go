package time

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
)

// TimeService is an implementation of domain.TimeService using the standard time package.
type TimeService struct{}

// Now returns the current time.
func (ts TimeService) Now() time.Time {
	return time.Now()
}

// InitTimeService initializes the TimeService and registers it in the dependency container.
type InitTimeService struct {
}

// Initialize registers the TimeService in the dependency container.
func (its InitTimeService) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.TimeService](TimeService{})
	return ctx, nil
}
