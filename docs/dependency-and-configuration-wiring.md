# Dependency and Configuration Wiring

Symbiont provides a simple, explicit mechanism for wiring **dependencies and configuration**
into initializers and runnables.

Wiring happens **before any component runs** and is based on declared struct fields,
not runtime lookups.

## Declaring Dependencies and Configuration

Dependencies and configuration are declared as struct fields using tags.

```go
type APIServer struct {
	Logger   *log.Logger `resolve:""`
	HttpPort int         `config:"HTTP_PORT" default:"80"`
}
```

At startup, Symbiont resolves dependencies from the container, reads configuration
values from the active provider, and injects them into the component before
execution begins.

If a dependency or required configuration value cannot be resolved, application
startup fails.

## Registering Dependencies

Dependencies are typically registered during initialization:

```go
type LoggerInitializer struct{}

func (i *LoggerInitializer) Initialize(ctx context.Context) (context.Context, error) {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	depend.Register[*log.Logger](logger)
	return ctx, nil
}
```

Registration is explicit and happens once, during initialization.

## Configuration Injection

Configuration values can be injected in the same way as dependencies.

This allows configuration to be treated as a first-class input to components,
rather than being accessed through global state or package-level variables.

### Wiring Guarantees

Symbiont guarantees that:

- all dependencies and configuration are resolved before any initializer or runnable executes
- components receive fully populated fields when they start
- wiring failures prevent the application from starting

This keeps component behavior explicit and predictable.
