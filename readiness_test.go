package symbiont

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eventuallyReady is a runnable that becomes ready after N calls
type eventuallyReady struct {
	readyAfter int
	calls      int
}

func (e *eventuallyReady) Run(ctx context.Context) error { <-ctx.Done(); return nil }
func (e *eventuallyReady) IsReady(ctx context.Context) error {
	e.calls++
	if e.calls >= e.readyAfter {
		return nil
	}
	return errors.New("not ready")
}

// alwaysNotReady is a runnable that never becomes ready
type alwaysNotReady struct{}

func (a *alwaysNotReady) Run(ctx context.Context) error     { <-ctx.Done(); return nil }
func (a *alwaysNotReady) IsReady(ctx context.Context) error { return errors.New("not ready") }

func TestWaitForReadiness(t *testing.T) {
	tests := map[string]struct {
		setup       func(*App)
		timeout     time.Duration
		ctxFn       func() (context.Context, context.CancelFunc)
		expectErr   bool
		expectErrIs error
		expectMsg   string
	}{
		"succeeds_before_timeout": {
			setup:   func(a *App) { a.Host(&eventuallyReady{readyAfter: 2}) },
			timeout: 500 * time.Millisecond,
			ctxFn: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			expectErr: false,
		},
		"times_out": {
			setup:   func(a *App) { a.Host(&alwaysNotReady{}) },
			timeout: 150 * time.Millisecond,
			ctxFn: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			expectErr: true,
			expectMsg: "not ready",
		},
		"context_canceled": {
			setup:   func(a *App) { a.Host(&alwaysNotReady{}) },
			timeout: 500 * time.Millisecond,
			ctxFn: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			expectErr:   true,
			expectErrIs: context.Canceled,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			a := NewApp()
			tt.setup(a)
			ctx, cancel := tt.ctxFn()
			defer cancel()

			// run the app asynchronously
			errCh := a.RunAsync(ctx)

			// wait for readiness
			err := a.WaitForReadiness(ctx, tt.timeout)

			if tt.expectErr {
				require.Error(t, err)
				if tt.expectErrIs != nil {
					assert.ErrorIs(t, err, tt.expectErrIs)
				}
				if tt.expectMsg != "" {
					var se *Error
					assert.True(t, errors.As(err, &se))
					assert.Contains(t, se.Error(), tt.expectMsg)
				}
			} else {
				require.NoError(t, err)
			}

			// cancel to stop the app and wait for it to complete
			cancel()
			select {
			case <-errCh:
				// ok
			case <-time.After(1 * time.Second):
				t.Fatal("RunAsync did not complete")
			}
		})
	}
}
