# Symbiont

A Go library for building modular, composable applications with lifecycle management, dependency injection, configuration management, and graceful shutdown.

## Overview

Symbiont provides a lightweight foundation for orchestrating complex applications by managing:

- **Initialization** — Sequential setup of components with context propagation
- **Dependency Injection** — Type-safe dependency resolution via struct tags
- **Configuration Management** — Flexible config value injection via struct tags
- **Readiness Polling** — Health checking before serving traffic
- **Graceful Shutdown** — LIFO resource cleanup with panic recovery
- **Error Propagation** — Context-aware error wrapping with component metadata

## Core Concepts

### Lifecycle

Every Symbiont application follows a simple lifecycle:

```
Initialize → Configure Dependencies → Host Runnables → Run → Cleanup
```

### Key Interfaces

**Initializer** — Setup phase for a component
```go
type Initializer interface {
    Initialize(context.Context) (context.Context, error)
}
```
- Called once at startup
- Can return a new context (e.g., with new values)
- Can register dependencies for later injection
- Errors halt the application

**Runnable** — Execution phase (long-lived task)
```go
type Runnable interface {
    Run(context.Context) error
}
```
- Runs concurrently after all initializers complete
- Receives injected dependencies and config
- Blocking; errors propagate via errgroup
- Context cancellation triggers graceful shutdown

**ReadyChecker** — Health checking (optional)
```go
type ReadyChecker interface {
    IsReady(context.Context) error
}
```
- Polled by `WaitForReadiness()` to detect startup completion
- Default implementation marks ready after `Run()` starts
- Enable load balancers to defer traffic until the app is ready

**Closer** — Cleanup phase (optional)
```go
type Closer interface {
    Close()
}
```
- Invoked in LIFO (reverse registration) order
- Blocks other closers; synchronize long operations externally
- Panics are recovered and logged via error wrapping

### Builder Pattern

Symbiont uses a fluent API for configuration:

```go
err := symbiont.NewApp().
    Initialize(&dbInit{}, &cacheInit{}, &{}loggerInit{}).
    Host(&httpServer{}, &grpcServer{}, &backgroundWorker{}).
    Run()
```

## Dependency Injection

Dependencies are registered during initialization and resolved into struct fields at runtime.

### How It Works

1. **Registration** — An Initializer calls `depend.Register[T](value)` to register a dependency
2. **Struct Tags** — Target types declare injectable fields with `resolve:""` tag
3. **Injection** — Before the Runnable executes, dependencies are resolved and injected into struct fields by type (configuration values are also injected at this same step)
4. **Access** — The Runnable reads resolved values from struct fields

### Example

```go
// Step 1: Initializer registers a database
type DbInit struct{
    db *sql.DB
    Logger *log.Logger
}
func (i *DbInit) Initialize(ctx context.Context) (context.Context, error) {
    db = openDatabase(ctx)
    depend.Register[*sql.DB](db)
    return ctx, nil
}

// Step 2: Runnable declares dependency via tag
type ApiServer struct {
    DB *sql.DB `resolve:""`  // Auto-injected before Run()
}
func (s *ApiServer) Run(ctx context.Context) error {
    return s.startServer(s.DB)  // DB is ready
}

// Step 3: Wire together
err := symbiont.NewApp().
    Initialize(&DbInit{}).
    Host(&ApiServer{}).
    Run()
```

### Rules

- **One per type** — Only one instance of each type can be registered
- **Pointer or value** — Can register `*sql.DB` or `sql.DB`, but not both
- **Early registration** — Must be registered during initialization phase
- **Type safety** — Resolution fails at runtime if type not registered; use generics for type checking

## Configuration Management

Configuration values are injected via a pluggable provider system.

### How It Works

1. **Provider** — Implement `config.Provider` interface or use built-in providers
2. **Registration** — Set global provider during initialization: `config.SetGlobalProvider(provider)`
3. **Struct Tags** — Target fields use `config:"keyName"` to request values
4. **Injection** — Before Runnable executes, config values are loaded and injected into struct fields
5. **Access** — Runnable reads injected values from struct fields

### Example

```go
// Step 1: Initializer sets a config provider
type ConfigInit struct {
    data map[string]string
}
func (c *ConfigInit) Initialize(ctx context.Context) (context.Context, error) {
    provider := &mapProvider{data: c.data}
    config.SetGlobalProvider(provider)
    return ctx, nil
}

// Step 2: Runnable declares config via tag
type AppService struct {
    Port   string `config:"port"`
    LogLevel string `config:"logLevel"`
}
func (s *AppService) Run(ctx context.Context) error {
    return s.startService(s.Port, s.LogLevel)
}

// Step 3: Wire together
app := symbiont.NewApp().
    Initialize(&ConfigInit{data: map[string]string{
        "port": "8080",
        "logLevel": "info",
    }}).
    Host(&AppService{}).
    Run()
```

### Built-in Support

Both dependency and configuration injection happen in the same injection phase:

- `depend.ResolveStructFieldValue` — Resolves `resolve:""` tags (dependencies)
- `config.LoadStructFieldValue(ctx)` — Loads `config:"key"` tags (configuration)
- Custom providers can implement any resolution logic

## Readiness Polling

`WaitForReadiness()` provides a health-check interface for integration tests.

### Default Behavior

Components that only implement `Runnable` get a default `ReadyChecker` that marks ready once `Run()` starts executing.

### Custom Ready Checks

Implement `ReadyChecker` on your component:

```go
type Server struct {
    started atomic.Bool
}

func (s *Server) Run(ctx context.Context) error {
    s.started.Store(true)
    return s.serve(ctx)
}

func (s *Server) IsReady(ctx context.Context) error {
    if !d.started.Load() {
        return errors.New("server not started")
    }
    // Check health: DB connectivity, dependencies, etc.
    if err := s.db.Ping(); err != nil {
        return err
    }
    return nil
}

// Usage
cancelCtx, cancel := context.WithCancel(context.Background())
errSig := symbiont.NewApp().Host(&apiServer{}).RunAnsync(cancelCtx)
if err := app.WaitForReadiness(context.Background(), 5*time.Second); err != nil {
    log.Fatal("app not ready:", err)
}

cancel()

select{
case err<-errSig:
    ...
case <-time.After(7 * time.Second):
    ...
}

```

### Polling

- Polls all `ReadyChecker`s every 50ms
- Returns early if all report ready
- Returns error if timeout elapses
- Respects context cancellation (e.g., from orchestrator shutdown signals)

## Error Handling

Symbiont wraps errors with contextual metadata for debugging.

### Error Type

```go
type Error struct {
    Err           error
    ComponentName string
    FileLine      string  // For functions
}

func (e *Error) Error() string {
    // e.g., "error: database connection failed, component: app.DbInit"
}
```

### When Errors Occur

- **Initialization** — If any Initializer returns error, app halts immediately
- **Injection** — If dependency/config resolution fails, app halts immediately
- **Panic Recovery** — Panics in Initialize/Run are caught and wrapped as errors
- **Runnable Errors** — Propagated via errgroup; first error returned

### Unwrapping

```go
var symErr *symbiont.Error
if errors.As(err, &symErr) {
    fmt.Printf("Failed component: %s\n", symErr.ComponentName)
    fmt.Printf("Original error: %v\n", symErr.Err)
}
```

## Graceful Shutdown

Symbiont ensures predictable cleanup via LIFO closer chains.

### How It Works

1. **Collection** — As Initializers and Runnables are set up, their `Close()` methods are collected
2. **Deferral** — All closers are deferred and invoked at exit
3. **LIFO Order** — Last registered closer is called first (reverse dependency order)
4. **No Synchronization** — Closers are called sequentially; block externally with `sync.Once` if needed

### Example

```go
type DbInit struct{}
func (d *DbInit) Close() { d.db.Close() }

type CacheInit struct{}
func (c *CacheClose) Close() { c.redis.Close() }

type ApiServer struct{}
func (s *ApiServer) Run(ctx context.Context) err { return s.httpServer.ListenAndServe() }
func (s *ApiServer) Close() { s.httpServer.Shutdown(shutdownCtx) }

// If registered as: Initialize(&DbClose{}, &CacheInit{}).Host(&ApiServer{})
// Cleanup order: ApiServer.Close, CacheInit.Close(), DbClose.Close()  (LIFO)
```

### Panic Recovery

Panics during cleanup are recovered and reported via error wrapping. The application exits with error status.

## Execution Flow

### 1. Run() / RunWithContext() / RunAsync()

**Run()** — Blocks until completion; handles OS signals (SIGINT, SIGTERM)

**RunWithContext()** — Blocks until completion; respects provided context

**RunAsync()** — Returns immediately; runs in background goroutine; returns error channel

All three invoke the same internal orchestrator:

### 2. Orchestration Phases

**Phase 1: Initialize**
- For each Initializer:
  - Inject dependencies and config into struct fields
  - Call `Initialize()`, which may return new context
  - Collect any `Close()` if implemented
  - Halt on error

**Phase 2: Wire Runnables**
- For each Runnable:
  - Inject dependencies and config into struct fields
  - Collect any `Close()` if implemented
  - Halt on error

**Phase 3: Execute**
- Launch each Runnable concurrently in an errgroup
- Each runs in a separate goroutine
- First error returned; others cancelled via context

**Phase 4: Cleanup**
- Collect all closers (from Phase 1 and 2)
- Call in LIFO order (reverse registration)
- Recover panics and report as errors
- Halt immediately on panic during cleanup

## Package Structure

### `symbiont` (main)
- **Types**: `App`, `Initializer`, `Runnable`, `Closer`, `ReadyChecker`
- **Functions**: `NewApp()`, `WaitForReadiness()`
- **Lifecycle Methods**: `Run()`, `RunWithContext()`, `RunAsync()`

### `symbiont/depend`
- **Functions**: `Register[T](value)`, `Resolve[T]()`, `ResolveStructFieldValue()`, `ClearContainer()`
- **Purpose**: Type-safe dependency container; called during initialization

### `symbiont/config`
- **Types**: `Provider` (interface)
- **Functions**: `SetGlobalProvider()`, `Get()`, `GetWithDefault()`, `LoadStructFieldValue()`, `ResetGlobalProvider()`
- **Purpose**: Configuration provider abstraction; loaded before runnables execute

### `symbiont/readiness` (in readiness.go/readiness_test.go)
- **Type**: `defaultReadyChecker` (internal)
- **Function**: `WaitForReadiness()` (on App)
- **Purpose**: Health checking and startup readiness polling

### `symbiont/error` (in error.go)
- **Type**: `Error` (exported)
- **Function**: `NewError()` (exported)
- **Purpose**: Context-aware error wrapping with component metadata

## Philosophy (Foundation for Refinement)

### Principles

1. **Composition** — Build apps by wiring small, focused components
2. **Explicitness** — Lifecycle phases are clear and predictable
3. **Type Safety** — Use generics for dependency resolution; catch misconfigurations early
4. **Fail Fast** — Errors halt initialization immediately; no silent failures
5. **Graceful Shutdown** — Cleanup happens predictably in reverse registration order
6. **Observability** — Errors include component context for debugging
7. **Flexibility** — Providers and tags allow multiple injection patterns

### Design Constraints

- **Sequential Initialization** — Simplifies context propagation and determinism
- **Concurrent Execution** — Runnables run in parallel for better resource utilization
- **LIFO Cleanup** — Mirrors dependency order; natural for resource hierarchies
- **No Global State Management** — Providers and containers are explicit; tests can isolate easily

## Quick Start

```go
package main

import (
    "context"
    "github.com/cleitonmarx/symbiont"
)

type MyService struct{}

func (s *MyService) Initialize(ctx context.Context) (context.Context, error) {
    println("Initializing...")
    return ctx, nil
}

func (s *MyService) Run(ctx context.Context) error {
    println("Running...")
    <-ctx.Done()
    return nil
}

func main() {
    app := symbiont.NewApp().
        Initialize(&MyService{}).
        Host(&MyService{})
    
    if err := app.Run(); err != nil {
        panic(err)
    }
}
```

---

*This documentation provides the foundation for future refinement. Topics for deeper exploration: advanced provider patterns, testing strategies, multi-stage initialization, metrics/observability hooks.*
