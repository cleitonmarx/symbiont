# Initializers and Runnables

Symbiont structures applications around two explicit component types:
**Initializers** and **Runnables**. Together, they define how an application is built
and how it runs.

## Initializers

Initializers are responsible for **startup logic**.

They run during the initialization phase, before any long-running services start.
Typical responsibilities include:

- loading configuration
- constructing shared dependencies
- registering resources in the dependency container
- enriching the application context

Initializers execute in the order they are registered.

An initializer implements a simple interface:

```go
type Initializer interface {
	Initialize(ctx context.Context) (context.Context, error)
}
```

Initializers may return an updated context, which is passed to subsequent initializers
and later to all runnables.

---

## Runnables

Runnables represent **long-lived services** that make up the runtime of the application.

They start after all initializers have completed successfully and are executed
concurrently. Typical examples include:

- HTTP servers
- background workers
- message consumers
- schedulers

A runnable implements the following interface:

```go
type Runnable interface {
	Run(ctx context.Context) error
}
```

The context passed to `Run` is cancelled when the application begins shutting down.
Runnables are expected to block until that context is cancelled and return cleanly.
