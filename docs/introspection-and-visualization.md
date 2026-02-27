# Introspection and Visualization

Symbiont exposes introspection capabilities that allow you to **inspect the application
structure it builds at runtime**, rather than treating it as a black box.

Introspection is **observational only**. It does not affect execution order,
lifecycle behavior, or dependency resolution.

It is primarily useful for:

- understanding how dependencies are wired
- validating application structure in tests
- debugging complex setups
- documenting and reviewing architectural decisions

## What Can Be Inspected

At runtime, Symbiont tracks:

- registered initializers and hosted runnables
- dependency registrations and resolutions
- configuration keys and providers used
- caller information (function and file location)

This information is aggregated into an introspection report once the application
lifecycle reaches the boundary between **wiring** and **execution**.

## Enabling Introspection

To enable introspection, register one or more introspectors on the application.

An introspector implements the following interface:

```go
type Introspector interface {
	Introspect(ctx context.Context, r introspection.Report) error
}
```

Introspectors are invoked **after initialization and wiring complete** — meaning all
initializers have executed and all dependency and configuration injection has been
resolved — **but before any runnables start executing**.

This places introspection at the boundary between **wiring** and **execution**, allowing
you to inspect the fully constructed application graph without affecting runtime behavior.

Each call to `Introspect(...)` appends a new introspector.

```go
app := symbiont.NewApp().
	Initialize(&LoggerInitializer{}).
	Introspect(&GraphLogger{}).
	Introspect(&AuditIntrospector{})
```

In addition, any hosted runnable that also implements `Introspector` is automatically
included in introspection.

```go
type WorkerWithIntrospection struct{}

func (w *WorkerWithIntrospection) Run(ctx context.Context) error { return nil }

func (w *WorkerWithIntrospection) Introspect(
	ctx context.Context,
	r introspection.Report,
) error {
	// inspect report
	return nil
}

app := symbiont.NewApp().
	Host(&WorkerWithIntrospection{})
```

## Generating Dependency Graphs (Mermaid)

Symbiont includes built-in support for generating **Mermaid diagrams** directly
from the introspection report.

A common and idiomatic pattern is to use an introspector with an **injected logger**
and log the generated Mermaid graph during startup introspection.

```go
type GraphLogger struct {
	Logger *log.Logger `resolve:""`
}

func (g *GraphLogger) Introspect(
	_ context.Context,
	r introspection.Report,
) error {
	graph := mermaid.GenerateIntrospectionGraph(r)
	g.Logger.Println(graph)
	return nil
}
```

Register introspectors like any other component:

```go
app := symbiont.NewApp().
	Initialize(&LoggerInitializer{}).
	Introspect(&GraphLogger{})
```

When introspection runs, the Mermaid graph is emitted to logs and
can be copied directly into Markdown, documentation, or review tools.

## Serving Mermaid Over HTTP

You can also serve an interactive Mermaid page using `mermaid.NewGraphHandler`.
The handler renders a snapshot of the provided report and serves pre-rendered HTML.

```go
package main

import (
	"net/http"

	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/cleitonmarx/symbiont/introspection/mermaid"
)

func registerIntrospectionRoute(mux *http.ServeMux, report introspection.Report) {
	mux.Handle(
		"/introspection",
		mermaid.NewGraphHandler(
			"My App",
			report,
			mermaid.WithMaxTextSize(100000), // optional, default is 100000
		),
	)
}
```

If `WithMaxTextSize(...)` is not provided, `100000` is used by default.

## Visualization (Mermaid)

The generated Mermaid graph visualizes:

- initializers and runnables
- dependencies and configuration providers
- relationships between components
- the overall shape of the application

Because the graph is derived from **runtime introspection data**, it always reflects
how the application actually runs, not how it is assumed to run.

Mermaid diagrams are especially useful for:

- onboarding new developers
- reviewing architectural changes
- understanding large or evolving systems


### Mermaid Example

The following diagram is an example of the Mermaid output style produced by Symbiont
introspection, including emoji labels, colored nodes, and relationship arrows.

```mermaid
---
config:
  layout: elk
---
graph TD
	cfg["<b><span style='font-size:16px'>cfg</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 provider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	DepImpl["<b><span style='font-size:16px'>Dep</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 DepImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ examples.(*initLogger).Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(f:1)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	ptr_examples_initLogger["<b><span style='font-size:16px'>*examples.initLogger</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	run1["<b><span style='font-size:16px'>run1</span></b><br/><span style='color:green;font-size:11px;'>⚙️ <b>Runnable</b></span>"]
	SymbiontApp["<b><span style='font-size:20px;color:white'>🚀 Symbiont App</span></b>"]
    ptr_examples_initLogger --o DepImpl
    DepImpl -.-> run1
    cfg -.-> ptr_examples_initLogger
    run1 --- SymbiontApp
    style cfg fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
    style DepImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
    style ptr_examples_initLogger fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
    style run1 fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
    style SymbiontApp fill:#0f56c4,stroke:#68a4eb,stroke-width:6px,color:#ffffff,font-weight:bold

```
