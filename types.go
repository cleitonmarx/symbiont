package symbiont

import "context"

// Runnable executes a long-lived process that runs concurrently with other runnables.
// The context is canceled on app shutdown or when the first runnable returns an error.
type Runnable interface {
	Run(context.Context) error
}

// Closer releases resources and is called during graceful shutdown.
// Closers are invoked in LIFO (reverse registration) order.
type Closer interface {
	Close()
}

// Initializer sets up component resources during application startup.
// It can register dependencies and return an updated context for propagation to other components.
// Errors halt initialization immediately; panics are recovered and reported.
type Initializer interface {
	Initialize(context.Context) (context.Context, error)
}

// ReadyChecker reports whether a runnable is ready to serve traffic.
// If not implemented, a default ReadyChecker marks ready once the runnable's Run method starts.
type ReadyChecker interface {
	IsReady(ctx context.Context) error
}
