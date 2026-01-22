<div align="center">
	<img src="assets/logo.png" style="width: 40%;">
</div>


# Symbiont

Symbiont is a lightweight **application host** for Go.

It provides a structured way to initialize dependencies, start long-lived services,
and coordinate application shutdown within a single, testable lifecycle.

---

## Why Symbiont Exists

Applications tend to accumulate complexity in `main.go`.

Startup logic, dependency wiring, background goroutines, and shutdown handling often end up
mixed together, making applications harder to read, reason about, and change.

Symbiont encourages structuring applications as explicit components with clear lifecycles
and dependencies. This improves organization and readability early on, and creates natural
boundaries that make it easier to reorganize the system — including splitting components
into separate deployables — as requirements grow.

## Mental Model

Symbiont hosts an application composed of initialization steps and long-running services,
managed under a single lifecycle.

That lifecycle follows a simple flow:

- **Initialization**: startup logic runs in a controlled sequence
- **Wiring**: dependencies and configuration are provided before any component runs
- **Execution**: long-running services start concurrently
- **Shutdown**: services stop and cleanup runs in a defined order

## Quick Start

Install Symbiont:

```shell
go get github.com/cleitonmarx/symbiont
```

A Symbiont application is composed of:

- **Initializers** — startup logic and dependency setup
- **Runnables** — long-lived services executed at runtime

### Minimal Example

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/cleitonmarx/symbiont"
	"github.com/cleitonmarx/symbiont/depend"
)

type LoggerInitializer struct{
	Prefix `config:"LOG_PREFIX" default:"app"`
}

func (i *LoggerInitializer) Initialize(ctx context.Context) (context.Context, error) {
	logger := log.New(os.Stdout, i.Prefix, log.LstdFlags)
	depend.Register[*log.Logger](logger)
	return ctx, nil
}

type Worker struct {
	Logger *log.Logger `resolve:""`
}

func (w *Worker) Run(ctx context.Context) error {
	w.Logger.Println("worker started")
	<-ctx.Done()
	w.Logger.Println("worker stopped")
	return nil
}

func main() {
	app := symbiont.NewApp().
		Initialize(&LoggerInitializer{}).
		Host(&Worker{})

	if err := app.Run(); err != nil {
		// Handle error
	}
}
```

---

## Documentation

Detailed documentation is available in the [`docs`](docs) directory:

- [Initializers and Runnables](docs/initializers-and-runnables.md)
- [Application Lifecycle](docs/lifecycle.md)
- [Dependency and Configuration Wiring](docs/dependency-and-configuration-wiring.md)
- [Running Applications](docs/running-applications.md)
- [Error Handling and Shutdown](docs/error-handling-and-shutdown.md)
- [Packages: depend and config](docs/packages-depend-and-config.md)
- [Introspection and Visualization](docs/introspection-and-visualization.md)

---

## Examples

The [`examples`](examples) directory contains programs demonstrating:

- application structure with initializers and runnables
- dependency and configuration wiring
- hosting multiple concurrent services
- readiness for integration-style testing
- introspection and Mermaid dependency graphs

Each example is self-contained and runnable.

---

## License

MIT
