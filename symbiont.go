// Package symbiont provides lifecycle management, dependency injection, and graceful shutdown for Go applications.
package symbiont

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"syscall"

	"github.com/cleitonmarx/symbiont/config"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/internal/reflectx"
	"github.com/cleitonmarx/symbiont/introspection"
	"golang.org/x/sync/errgroup"
)

// runnableSpecs bundles a runnable with its executor and ready checker.
// The executor may wrap the original runnable with a default ready checker.
type runnableSpecs struct {
	// executor is the runnable that will be executed (may be wrapped)
	executor Runnable
	// original is the user-provided runnable
	original Runnable
	// readyChecker is the health check for this runnable
	readyChecker ReadyChecker
}

// closerFunc is a function that performs cleanup operations.
type closerFunc func()

// App orchestrates application lifecycle: initialization, concurrent execution, and graceful shutdown.
type App struct {
	initializers      []Initializer
	runnableSpecsList []runnableSpecs
	introspector      Introspector
	errCh             chan error
}

// NewApp creates a new application with no initializers or runnables.
func NewApp() *App {
	return &App{}
}

// Initialize adds initializers to the app (fluent method).
// Initializers run sequentially before runnables; use this to set up resources and register dependencies.
func (a *App) Initialize(init ...Initializer) *App {
	a.initializers = append(a.initializers, init...)
	return a
}

// Host adds runnables to the app (fluent method).
// Runnables execute concurrently after all initializers complete.
func (a *App) Host(runnable ...Runnable) *App {
	for _, r := range runnable {
		var (
			readyChecker ReadyChecker
			executor     Runnable
		)
		if rc, ok := r.(ReadyChecker); ok {
			readyChecker = rc
			executor = r
		} else {
			rc := &defaultReadyChecker{
				runable: r,
			}
			executor = rc
			readyChecker = rc
		}

		a.runnableSpecsList = append(a.runnableSpecsList, runnableSpecs{
			original:     r,
			executor:     executor,
			readyChecker: readyChecker,
		})
	}
	return a
}

// Run executes the app: initializes components, runs runnables concurrently, and handles graceful shutdown.
// Blocks until completion or signal (SIGINT, SIGTERM). Returns error if any phase fails.
func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		// Interrupt signal sent from terminal
		os.Interrupt,
		// Termination signal sent from Kubernetes or other orchestrators
		syscall.SIGTERM,
	)
	defer stop()

	return a.runWithContext(ctx)

}

// RunWithContext executes the app with the provided context for cancellation control.
func (a *App) RunWithContext(ctx context.Context) error {
	return a.runWithContext(ctx)
}

// RunAsync executes the app asynchronously in a background goroutine.
// Returns a channel that receives the final error (or nil) when execution completes.
func (a *App) RunAsync(ctx context.Context) chan error {
	a.errCh = make(chan error, 1)
	go func() {
		a.errCh <- a.runWithContext(ctx)
		close(a.errCh)
	}()
	return a.errCh
}

// runWithContext is the core orchestrator: initializes, wires dependencies, runs runnables, cleans up.
func (a *App) runWithContext(ctx context.Context) error {
	var closers []closerFunc
	defer func() { combineClosers(closers)() }()

	// Initialize all initializers and collect their closers
	for _, initializer := range a.initializers {
		err := wireStructFields(ctx, initializer)
		if err != nil {
			return err
		}

		newCtx, err := initializeSafe(ctx, initializer)
		if err != nil {
			return err
		}
		if newCtx != nil {
			ctx = newCtx
		}
		if closer, ok := initializer.(Closer); ok {
			if closer != nil {
				closers = append(closers, closer.Close)
			}
		}
	}

	// Load configuration and dependencies into all hosted runnables and collect their closers
	for _, rs := range a.runnableSpecsList {
		err := wireStructFields(ctx, rs.original)
		if err != nil {
			return err
		}
		if closer, ok := rs.original.(Closer); ok {
			if closer != nil {
				closers = append(closers, closer.Close)
			}
		}
	}

	// Call introspector if configured
	if a.introspector != nil {
		if err := wireStructFields(ctx, a.introspector); err != nil {
			return err
		}
		report := introspection.Report{
			Configs:      config.IntrospectConfigAccesses(),
			Deps:         depend.GetEvents(),
			Runners:      a.runnerInfos(),
			Initializers: a.initializerInfos(),
		}
		err := introspectSafe(ctx, a.introspector, report)
		if err != nil {
			return err
		}
	}

	// Run all hosted runnables
	errGroup, groupCtx := errgroup.WithContext(ctx)
	for _, rs := range a.runnableSpecsList {
		func(r runnableSpecs) {
			errGroup.Go(func() error {
				if err := runSafe(groupCtx, r); err != nil {
					return err
				}
				return nil
			})
		}(rs)
	}

	return errGroup.Wait()
}

// combineClosers returns a function that invokes all closers in LIFO (reverse) order.
// Captures the closers slice at defer time for consistent cleanup order.
func combineClosers(closers []closerFunc) closerFunc {
	return func() {
		for i := len(closers) - 1; i >= 0; i-- {
			closers[i]()
		}
	}
}

func (a *App) runnerInfos() []introspection.RunnerInfo {
	rInfos := make([]introspection.RunnerInfo, 0, len(a.runnableSpecsList))
	for _, rs := range a.runnableSpecsList {
		t := reflect.TypeOf(rs.original)
		rInfos = append(rInfos, introspection.RunnerInfo{
			Type:      reflectx.GetTypeName(t),
			Component: t,
		})
	}
	return rInfos
}

func (a *App) initializerInfos() []introspection.InitializerInfo {
	inits := make([]introspection.InitializerInfo, 0, len(a.initializers))
	for _, init := range a.initializers {
		t := reflect.TypeOf(init)
		inits = append(inits, introspection.InitializerInfo{
			Type:      reflectx.GetTypeName(t),
			Component: t,
		})
	}
	return inits
}

// initializeSafe calls an initializer's Initialize method with panic recovery.
// Returns the updated context and wraps both panics and errors in NewError.
func initializeSafe(ctx context.Context, init Initializer) (newCtx context.Context, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = NewError(fmt.Errorf("panic in Initialize func: %v", r), init)
		}
	}()
	newCtx, err = init.Initialize(ctx)
	if err != nil {
		err = NewError(err, init.Initialize)
	}
	return newCtx, err
}

// runSafe calls a runnable's Run method with panic recovery.
// Wraps both panics and errors in NewError for debugging.
func runSafe(ctx context.Context, rs runnableSpecs) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = NewError(fmt.Errorf("panic in Run func: %v", r), rs.original)
		}
	}()
	err = rs.executor.Run(ctx)
	if err != nil {
		err = NewError(err, rs.original.Run)
	}
	return err
}

// wireStructFields injects dependencies and configuration into struct fields via tags.
// Resolves resolve:"name" tags for dependencies and config:"key" tags for configuration.
func wireStructFields(ctx context.Context, target any) error {
	err := reflectx.IterateStructFields(
		target,
		depend.ResolveStructFieldValue,
		config.LoadStructFieldValue(ctx),
	)

	if err != nil {
		return NewError(err, target)
	}
	return nil
}
