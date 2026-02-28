# Running Applications

Symbiont applications are driven by a **context**. That context defines when the application
should keep running and when it should begin shutting down.

## Context Cancellation

Both `Run` and `RunAsync` derive application lifetime from a context. When that context is
canceled, the application begins a graceful shutdown.

Typical cancellation sources include:

- explicit cancellation by the caller
- errors returned by runnables
- external termination signals

Runnables are expected to observe the context passed to `Run` and return when it is canceled.

## Signal Handling

When using `Run`, Symbiont installs signal handlers for common termination signals,
including:

- `os.Interrupt`
- `syscall.SIGTERM`

Receiving one of these signals triggers context cancellation and starts the shutdown
sequence.

This allows applications to terminate cleanly without custom signal handling code
in `main`.

## Error Propagation

If a runnable returns an error during execution, the application initiates shutdown.

- with `Run`, the error is returned to the caller
- with `RunAsync`, the error is delivered through `shutdownCh`

This ensures that failures in any runnable are surfaced and handled consistently.

## Explicit Shutdown

When using `RunAsync`, the caller is responsible for initiating shutdown, typically by
canceling the context used to start the application.

```go
ctx, cancel := context.WithCancel(context.Background())
shutdownCh := app.RunAsync(ctx)

// later
cancel()
err := <-shutdownCh
```

This gives tests and embedded scenarios precise control over application lifetime.
