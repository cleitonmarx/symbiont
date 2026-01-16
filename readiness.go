package symbiont

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

// defaultReadyChecker is a default implementation of the ReadyChecker interface.
// It marks the task as ready after the Run method has been called.
type defaultReadyChecker struct {
	started atomic.Bool
	runable Runnable
}

func (d *defaultReadyChecker) Run(ctx context.Context) error {
	d.started.Store(true)
	return d.runable.Run(ctx)
}

func (d *defaultReadyChecker) IsReady(ctx context.Context) error {
	if d.started.Load() {
		return nil
	}
	return errors.New("not ready")
}

// WaitForReadiness polls all hosted runnables that implement the ReadyChecker interface until
// either all of them report ready, the provided timeout elapses, or the context is canceled.
//
// If all ready checkers become ready before the timeout, it returns nil. If the context is
// canceled, it returns the context's error. If the timeout elapses and some runnable is still
// not ready, it returns the last readiness error wrapped with the failing component via NewError.
func (a *App) WaitForReadiness(ctx context.Context, timeout time.Duration) error {
	// collect ready checkers
	if len(a.runnableSpecsList) == 0 {
		return nil
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	var lastFailing any

	for {
		// check all
		allReady := true
		for _, c := range a.runnableSpecsList {
			if err := c.readyChecker.IsReady(waitCtx); err != nil {
				lastErr = err
				lastFailing = c.original
				allReady = false
				break
			}
		}
		if allReady {
			return nil
		}

		select {
		case err := <-a.errCh:
			// If the app has stopped running, return its final error
			return err
		case <-waitCtx.Done():
			// If the parent context was canceled, prefer returning that cancellation error.
			if waitCtx.Err() == context.Canceled {
				return waitCtx.Err()
			}
			// If the timeout elapsed (deadline exceeded), return the last readiness error wrapped
			// with the failing component when available; otherwise return the context error.
			if waitCtx.Err() == context.DeadlineExceeded {
				if lastFailing != nil && lastErr != nil {
					return NewError(lastErr, lastFailing)
				}
				return waitCtx.Err()
			}
			// fallback: return the context error
			return waitCtx.Err()
		case <-ticker.C:
			// try again
		}
	}
}
