package symbiont

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
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
	return errors.New("not ready yet")
}

// alwaysNotReady is a runnable that never becomes ready
type alwaysNotReady struct{}

func (a *alwaysNotReady) Run(ctx context.Context) error     { <-ctx.Done(); return nil }
func (a *alwaysNotReady) IsReady(ctx context.Context) error { return errors.New("never ready") }

// immediatelyReady is a runnable that is always ready
type immediatelyReady struct{}

func (i *immediatelyReady) Run(ctx context.Context) error     { <-ctx.Done(); return nil }
func (i *immediatelyReady) IsReady(ctx context.Context) error { return nil }

func TestWaitForReadiness(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*App)
		runnables    []Runnable
		timeout      time.Duration
		ctxFn        func() (context.Context, context.CancelFunc)
		expectErr    bool
		expectErrIs  error
		expectMsg    string
		expectSymErr bool
	}{
		{
			name:      "no-runnables",
			setup:     nil,
			runnables: nil,
			timeout:   50 * time.Millisecond,
			ctxFn: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 100*time.Millisecond)
			},
			expectErr: false,
		},
		{
			name:      "all-ready-immediately",
			setup:     nil,
			runnables: []Runnable{&immediatelyReady{}},
			timeout:   50 * time.Millisecond,
			ctxFn: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 100*time.Millisecond)
			},
			expectErr: false,
		},
		{
			name:      "eventually-ready",
			setup:     nil,
			runnables: []Runnable{&eventuallyReady{readyAfter: 2}},
			timeout:   100 * time.Millisecond,
			ctxFn: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 100*time.Millisecond)
			},
			expectErr: false,
		},
		{
			name:      "never-ready",
			setup:     nil,
			runnables: []Runnable{&alwaysNotReady{}},
			timeout:   10 * time.Millisecond,
			ctxFn: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 100*time.Millisecond)
			},
			expectErr: true,
		},
		{
			name:    "succeeds-before-timeout",
			setup:   func(a *App) { a.Host(&eventuallyReady{readyAfter: 2}) },
			timeout: 500 * time.Millisecond,
			ctxFn: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			expectErr: false,
		},
		{
			name:    "times-out",
			setup:   func(a *App) { a.Host(&alwaysNotReady{}) },
			timeout: 150 * time.Millisecond,
			ctxFn: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			expectErr: true,
			expectMsg: "never ready",
		},
		{
			name:    "context-canceled",
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewApp()
			if tt.setup != nil {
				tt.setup(a)
			}
			for _, r := range tt.runnables {
				a = a.Host(r)
			}
			ctx, cancel := tt.ctxFn()
			defer cancel()

			// run the app asynchronously
			errCh := a.RunAsync(ctx)

			// wait for readiness
			err := a.WaitForReadiness(ctx, tt.timeout)

			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.expectErrIs != nil {
					if !errors.Is(err, tt.expectErrIs) {
						t.Fatalf("expected error to match %v, got %v", tt.expectErrIs, err)
					}
				}
				if tt.expectMsg != "" {
					var se Error
					if !errors.As(err, &se) {
						t.Fatalf("expected symbiont.Error, got %T", err)
					}
					if !strings.Contains(se.Error(), tt.expectMsg) {
						t.Fatalf("expected error to contain %q, got %q", tt.expectMsg, se.Error())
					}
				}
				if tt.expectSymErr {
					var se Error
					if !errors.As(err, &se) {
						t.Fatalf("expected symbiont.Error wrapper, got %T", err)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
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
