# Application Lifecycle

A Symbiont application is executed by calling either `Run` or `RunAsync` on the app.

These methods control **how the caller interacts with the lifecycle**, not how components
are structured or executed.

## Run

`Run` starts the application and blocks until it terminates.

It is the most common entry point and is typically used in production binaries.

```go
err := app.Run()
```

When `Run` is called:

1. All initializers execute
2. All runnables start concurrently
3. The call blocks until shutdown completes
4. Any error is returned to the caller

---

## RunAsync

`RunAsync` starts the application without blocking the caller.

```go
shutdownCh := app.RunAsync(ctx)
```

This method is primarily intended for **integration tests** and advanced embedding
scenarios where the caller needs explicit control over the application lifecycle.

`RunAsync` returns a `shutdownCh` that is used to observe termination. When the
application shuts down:

- the application error (if any) is sent through `shutdownCh`
- the channel is then closed

The caller may:

- cancel the provided context to initiate shutdown
- receive from `shutdownCh` to obtain the application error
- wait for `shutdownCh` to close to know shutdown is complete

The execution of initializers and runnables is identical to `Run`; only the
blocking behavior differs.

## Integration Testing and RunAsync

When running applications asynchronously, callers often need to know **when the system
is usable**, not just when it has been started.

For this purpose, Symbiont provides a readiness utility intended for **integration tests**
and controlled embedding scenarios.

### Readiness

Readiness allows a test or caller to wait until hosted runnables report they are ready.
It does **not**:

- control execution order
- gate service startup
- influence runtime behavior

All initializers and runnables are executed as usual; readiness is purely an observation
mechanism.

### ReadyChecker and the Default Behavior

Runnables may optionally implement `ReadyChecker`:

```go
type ReadyChecker interface {
	IsReady(ctx context.Context) error
}
```

If a runnable does **not** implement `ReadyChecker`, Symbiont applies a default checker.
That default checker reports ready **as soon as the runnable's `Run()` method has been called**
(i.e., once the runnable goroutine has started executing).

This means:

- runnables with a custom `IsReady` can expose real readiness (e.g., "server is listening")
- runnables without it are considered ready after they start running

### Typical Usage in Tests

```go
shutdownCh := app.RunAsync(ctx)

if err := app.WaitForReadiness(ctx, 10*time.Second); err != nil {
	t.Fatal(err)
}

// run test assertions here

cancel() // cancel the context used by RunAsync
err := <-shutdownCh
if err != nil {
	t.Fatal(err)
}
```

Notes:

- `WaitForReadiness` polls readiness until all hosted runnables are ready, the timeout elapses,
  the context is canceled, or the application stops.
- If the app stops while waiting, `WaitForReadiness` returns the application's final error.
