# Symbiont

Symbiont is a lightweight Go library for building modular, composable applications with lifecycle management, dependency injection, configuration management, readiness polling, and graceful shutdown.

Symbiont helps you initialize dependencies, host concurrent runnables, and coordinate cleanup into a cohesive, testable system.

---

## Table of Contents

- [Symbiont](#symbiont)
  - [Table of Contents](#table-of-contents)
  - [Why Symbiont?](#why-symbiont)
  - [Quick Start](#quick-start)
      - [Example](#example)
  - [Features Table](#features-table)
  - [Philosophy](#philosophy)
  - [Package Overview](#package-overview)
  - [Core Concepts](#core-concepts)
  - [Initializer](#initializer)
      - [Example](#example-1)
  - [Runnable](#runnable)
  - [Struct Injection](#struct-injection)
      - [Example](#example-3)
  - [Dependency Management Patterns](#dependency-management-patterns)
    - [Ordered Initialization](#ordered-initialization)
      - [Declaring Dependencies](#declaring-dependencies)
      - [Single Dependency Example](#single-dependency-example)
      - [Multiple Dependencies Example](#multiple-dependencies-example)
      - [Dependency Order Diagram](#dependency-order-diagram)
    - [Concurrent Hosting](#concurrent-hosting)
      - [Example](#example-4)
  - [Readiness Checks](#readiness-checks)
    - [ReadyChecker Interface](#readychecker-interface)
      - [Default Readiness](#default-readiness)
      - [Example: Service implementing ReadyChecker](#example-service-implementing-readychecker)
    - [WaitForReadiness Method](#waitforreadiness-method)
      - [Example: Wait for all](#example-wait-for-all)
      - [Example: Wait for specific services](#example-wait-for-specific-services)
  - [Running Your Application](#running-your-application)
      - [Example: Using `.RunAsync()` and `.WaitForReadiness()` in a test](#example-using-runasync-and-waitforreadiness-in-a-test)
  - [Error Handling \& Panic Recovery](#error-handling--panic-recovery)
      - [Example](#example-5)
  - [Context Propagation \& Cleanup](#context-propagation--cleanup)
  - [Dependency Injection (`depend`)](#dependency-injection-depend)
    - [How to Register/Resolve a Dependency](#how-to-registerresolve-a-dependency)
  - [Configuration (`config`)](#configuration-config)
      - [Example](#example-6)
  - [Introspection](#introspection)
    - [Features](#features)
    - [Usage](#usage)
      - [Example: Capture config keys and dependency events](#example-capture-config-keys-and-dependency-events)
    - [When to Use](#when-to-use)
  - [Testing Support](#testing-support)
  - [Examples](#examples)

---

## Why Symbiont?

Go applications often grow into large, tightly coupled `main` functions that mix initialization, configuration, and runtime logic. Symbiont reduces that complexity with a structured lifecycle, typed dependency injection, and predictable cleanup, making your codebase easier to maintain and test.

---

## Quick Start

Install Symbiont:

```bash
go get github.com/cleitonmarx/symbiont
```

#### Example

Use separate structs for initialization and execution:

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/cleitonmarx/symbiont"
    "github.com/cleitonmarx/symbiont/depend"
)

type LoggerInit struct{}

func (l *LoggerInit) Initialize(ctx context.Context) (context.Context, error) {
    logger := log.New(os.Stdout, "", log.LstdFlags)
    depend.Register[*log.Logger](logger)
    return ctx, nil
}

type Worker struct{
    Logger *log.Logger `resolve:""`
}

func (w *Worker) Run(ctx context.Context) error {
    w.Logger.Println("Running...")
    <-ctx.Done()
    return nil
}

func main() {
    app := symbiont.NewApp().
        Initialize(&LoggerInit{}).
        Host(&Worker{})

    if err := app.Run(); err != nil {
        panic(err)
    }
}
```

---

## Features Table

| Feature                        | Description                                                      |
|--------------------------------|------------------------------------------------------------------|
| Modular Initialization         | Sequential setup with context propagation                        |
| Unified Injection              | Dependencies and config injected via struct tags                 |
| Type-Safe DI                   | Generic registration and resolution in `depend`                  |
| Readiness Polling              | `WaitForReadiness` for startup coordination                      |
| Graceful Shutdown              | LIFO cleanup with panic recovery                                 |
| Error Context                  | Wrapped errors include component names and locations             |
| Introspection                  | Track used config keys and dependency events                     |

---

## Philosophy

Symbiont promotes composition over monoliths: small components with clear lifecycle roles that can be wired together deterministically. Initialization is sequential, execution is concurrent, and cleanup is predictable. This model keeps startup logic testable and makes runtime behavior easier to reason about.

---

## Package Overview

- `symbiont`: Core lifecycle (`NewApp`, `Initialize`, `Host`, `Run`, `RunWithContext`, `RunAsync`, `WaitForReadiness`)
- `depend`: Type-safe dependency injection container
- `config`: Configuration management with provider support and struct tags

---

## Core Concepts

Symbiont separates **initialization** from **execution** and uses optional interfaces for readiness and cleanup:

| Concept          | Purpose                                        |
|-----------------|------------------------------------------------|
| **Initializer** | Setup logic; can register dependencies and return a new context |
| **Runnable**    | Long-lived work; runs concurrently after initialization          |
| **ReadyChecker**| Optional readiness signal for `WaitForReadiness`                 |
| **Closer**      | Optional cleanup hook executed in LIFO order                     |

---

## Initializer

An `Initializer` is used for setup, dependency registration, and configuration providers.

```go
type Initializer interface {
    Initialize(ctx context.Context) (context.Context, error)
}
```

- Runs sequentially before any runnables
- Can register dependencies or set configuration providers
- Can return a new context for downstream components
- Errors halt startup immediately

#### Example

```go
type ConfigInit struct{}

func (c *ConfigInit) Initialize(ctx context.Context) (context.Context, error) {
    config.SetGlobalProvider(config.NewEnvVarProvider())
    return ctx, nil
}
```

---

## Runnable

A `Runnable` is a hosted component that runs concurrently after all initializers complete.

```go
type Runnable interface {
    Run(context.Context) error
}
```

- Runnables start after initialization
- All runnables execute concurrently
- Errors propagate via `errgroup` and cancel the app context

**Example:**

```go
type Job struct{}

func (j *Job) Run(ctx context.Context) error {
    // Background work
    <-ctx.Done()
    return nil
}
```

---

## Struct Injection

Symbiont wires struct fields automatically during both initialization and execution. It injects:

- Dependencies via `resolve:"name"` tags
- Config values via `config:"KEY"` tags (with optional `default:"value"`)

Wiring happens during initialization and before runnables start.

#### Example

```go
type MyService struct {
    DB   *sql.DB `resolve:""`
    Port int     `config:"APP_PORT" default:"8080"`
}
```

---

## Dependency Management Patterns

Symbiont relies on explicit initialization order and dependency injection to coordinate components.

### Ordered Initialization

Register initializers in the order they must execute.

#### Declaring Dependencies

Use dependency injection: an initializer registers dependencies and runnables resolve them via struct tags.

#### Single Dependency Example

```go
type DbInit struct{}

func (d *DbInit) Initialize(ctx context.Context) (context.Context, error) {
    db := openDatabase()
    depend.Register[*sql.DB](db)
    return ctx, nil
}
```

#### Multiple Dependencies Example

```go
type Api struct {
    DB *sql.DB `resolve:""`
}
```

#### Dependency Order Diagram

```
ConfigInit -> DbInit -> Hosted Runnables
```

---

### Concurrent Hosting

All runnables start concurrently once initialization finishes.

#### Example

```go
app := symbiont.NewApp().
    Initialize(&DbInit{}).
    Host(&ApiServer{}, &Worker{})
```

---

## Readiness Checks

Readiness polling is available for all hosted runnables.

### ReadyChecker Interface

```go
type ReadyChecker interface {
    IsReady(ctx context.Context) error
}
```

#### Default Readiness

If a runnable does not implement `ReadyChecker`, Symbiont wraps it in a default checker that marks ready once `Run` starts.

#### Example: Service implementing ReadyChecker

```go
type Server struct {
    started atomic.Bool
}

func (s *Server) Run(ctx context.Context) error {
    s.started.Store(true)
    <-ctx.Done()
    return nil
}

func (s *Server) IsReady(ctx context.Context) error {
    if s.started.Load() {
        return nil
    }
    return errors.New("not ready")
}
```

### WaitForReadiness Method

```go
func (a *App) WaitForReadiness(ctx context.Context, timeout time.Duration) error
```

- Waits for all hosted runnables to report ready
- Returns an error on timeout or cancellation

#### Example: Wait for all

```go
if err := app.WaitForReadiness(ctx, 5*time.Second); err != nil {
    log.Fatalf("readiness failed: %v", err)
}
```

---

## Running Your Application

Symbiont provides three main entry points: `.Run()`, `.RunWithContext()`, and `.RunAsync()`.

- `.Run()` starts with a signal-aware context (SIGINT/SIGTERM)
- `.RunWithContext(ctx)` runs with your own context
- `.RunAsync(ctx)` runs in a goroutine and returns an error channel

#### Example: Using `.RunAsync()` and `.WaitForReadiness()` in a test

```go
func TestAppLifecycle(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    app := symbiont.NewApp().
        Initialize(&ConfigInit{}).
        Host(&Server{})

    errCh := app.RunAsync(ctx)

    if err := app.WaitForReadiness(ctx, 5*time.Second); err != nil {
        t.Fatalf("readiness failed: %v", err)
    }

    cancel()

    select {
    case err := <-errCh:
        if err != nil {
            t.Fatalf("shutdown error: %v", err)
        }
    case <-time.After(5 * time.Second):
        t.Fatal("timeout waiting for shutdown")
    }
}
```

---

## Error Handling & Panic Recovery

Symbiont wraps errors and panics from `Initialize` and `Run` in `*symbiont.Error`, including component metadata for easier debugging.

#### Example

```go
if err := app.Run(); err != nil {
    var se *symbiont.Error
    if errors.As(err, &se) {
        fmt.Printf("component: %s\n", se.ComponentName)
        fmt.Printf("location: %s\n", se.FileLine)
        fmt.Printf("error: %v\n", se.Err)
    }
}
```

---

## Context Propagation & Cleanup

- Each `Initializer` can return a new context to be passed to later initializers and all runnables.
- Any component implementing `Closer` will have `Close()` called during shutdown.
- Cleanup runs in LIFO order; panics are recovered and wrapped as errors.

---

## Dependency Injection (`depend`)

The `depend` package provides a type-safe dependency injection container.

### How to Register/Resolve a Dependency

```go
// Register a dependency
depend.Register[*sql.DB](db)

// Ensure only one instance is registered
err := depend.RegisterOnce[*sql.DB](db)

// Register a named dependency
depend.RegisterNamed[*sql.DB](db, "mydb")

// Ensure only one named instance
err := depend.RegisterNamedOnce[*sql.DB](db, "mydb")

// Resolve a dependency
db, err := depend.Resolve[*sql.DB]()

// Resolve a named dependency
db, err := depend.ResolveNamed[*sql.DB]("mydb")

// Resolve dependencies into a struct
type MyService struct {
    DB      *sql.DB `resolve:""`
    NamedDB *sql.DB `resolve:"mydb"`
}
var svc MyService
if err := depend.ResolveStruct(&svc); err != nil {
    return err
}
```

---

## Configuration (`config`)

The `config` package provides a pluggable configuration system with struct tag injection.

#### Example

```go
// Get a configuration value by key
val, err := config.Get[string](ctx, "APP_PORT")

// Get a configuration value with a default
val := config.GetWithDefault[int](ctx, "APP_PORT", 8080)

// Load a struct with configuration values
var cfg = struct {
    Port int `config:"APP_PORT" default:"8080"`
    DSN  string `config:"DB_DSN"`
}{}

if err := config.LoadStruct(ctx, &cfg); err != nil {
    return err
}

// Register a custom parser
config.RegisterParser[[]string](func(value string) ([]string, error) {
    return strings.Split(value, ","), nil
})

stringSlice, err := config.Get[[]string](ctx, "MY_LIST")

// Custom provider
config.SetGlobalProvider(config.NewEnvVarProvider())
```

---

## Introspection

Symbiont exposes an `Instrospect` hook to capture which config keys and dependencies were used.

### Features

- **Dependency Events:** Inspect registration and resolution events from `depend`
- **Configuration Usage:** See which config keys were accessed and by which provider

### Usage

Implement the `Introspector` interface and register it with `App.Instrospect`:

#### Example: Capture config keys and dependency events

```go
type myIntrospector struct{}

func (myIntrospector) Introspect(_ context.Context, ai symbiont.AppIntrospection) error {
    for _, key := range ai.Keys {
        fmt.Printf("key: %s provider: %s default: %v caller: %s\n",
            key.Key, key.Provider, key.UsedDefault, key.Caller.Func)
    }
    for _, event := range ai.Events {
        fmt.Printf("dep: %s action: %s caller: %s\n",
            event.Type, event.Kind, event.Caller.Func)
    }
    return nil
}

app := symbiont.NewApp().
    Instrospect(&myIntrospector{})
```

### When to Use

- During development to validate wiring
- In tests to assert config usage
- For debugging dependency registration and resolution

---

## Testing Support

- Use `depend.ClearContainer()` to isolate dependency state between tests
- Use `config.ResetGlobalProvider()` to reset configuration providers
- Use `RunAsync(ctx)` to run the app in the background during tests
- Use `WaitForReadiness(ctx, timeout)` to wait for startup completion

---

## Examples

- See `symbiont_test.go` and `readiness_test.go` for runnable examples and readiness patterns.

---
