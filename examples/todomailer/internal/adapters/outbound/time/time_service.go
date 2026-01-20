package time

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
)

type TimeService struct{}

func (ts TimeService) Now() time.Time {
	return time.Now()
}

func (ts TimeService) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.TimeService](ts)
	return ctx, nil
}

type InitTimeService struct {
}

func (its *InitTimeService) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.TimeService](TimeService{})
	return ctx, nil
}
