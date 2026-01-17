# Error Handling and Shutdown

Symbiont defines a clear and deterministic shutdown process to ensure resources are
released safely and consistently.

## Shutdown Triggers

Shutdown may be initiated by:

- context cancellation
- an error returned by any runnable
- OS termination signals (when using `Run`)

Regardless of the trigger, shutdown follows the same sequence.

## The Closer Interface

Initializers and runnables may optionally implement `Closer` to participate in shutdown:

```go
type Closer interface {
	Close(ctx context.Context)
}
```

When a component implements `Closer`, Symbiont invokes `Close` during shutdown to allow
the component to release resources (for example, closing connections, stopping servers,
or flushing buffers).

`Close` does **not** return an error. Handling failures during cleanup is the
responsibility of the component itself (e.g., logging or metrics).

## Shutdown Sequence

When shutdown begins:

1. The application context is canceled
2. Runnables are expected to observe the cancellation and return
3. `Close(ctx)` is invoked for components that implement `Closer`
4. The application terminates with a final error (if any)

This ensures shutdown behavior is predictable and does not depend on how termination
was initiated.

## Close Ordering

`Close` is executed in **reverse order of registration and hosting**.

This allows dependencies to be torn down safely, mirroring how they were created
during initialization.

## Error Semantics

Errors that initiate shutdown are preserved and reported:

- with `Run`, the error is returned to the caller
- with `RunAsync`, the error is delivered through `shutdownCh`

Cleanup errors do not affect the final application error.