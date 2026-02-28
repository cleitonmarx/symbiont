package symbiont

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont/introspection"
)

// Introspector defines an interface for introspecting application runners, configuration and dependencies.
type Introspector interface {
	Introspect(context.Context, introspection.Report) error
}

// Introspect registers an introspector for the application's lifecycle.
// Multiple calls append introspectors in registration order.
// The provided Introspector's Introspect method will be called after initialization
// and before starting runnables.
func (a *App) Introspect(i Introspector) *App {
	if i == nil {
		return a
	}
	a.introspectors = append(a.introspectors, i)
	return a
}

// introspectSafe calls the provided Introspector's Introspect method safely,
// recovering from panics and wrapping errors with context about the introspector.
func introspectSafe(ctx context.Context, i Introspector, r introspection.Report) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = NewError(fmt.Errorf("panic in Introspect func: %v", r), i)
		}
	}()
	err = i.Introspect(ctx, r)
	if err != nil {
		err = NewError(err, i)
	}
	return err
}
