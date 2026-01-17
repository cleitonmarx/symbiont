# Packages: depend and config

Symbiont includes two supporting packages that handle **dependency wiring** and
**configuration management**. Both are built on **Go generics** to provide
type safety and early validation.

These packages are used internally by the application host, but are also designed
to be usable independently.

---

## Package `depend`

The `depend` package provides a **type-safe dependency container**.

It is responsible for:

- registering concrete values by type
- resolving dependencies into components
- detecting missing or duplicate registrations
- enforcing wiring correctness at startup time

All dependency registration and resolution happens **before any initializer or
runnable executes**.

### Registering Dependencies

Dependencies are registered explicitly, typically during initialization:

```go
depend.Register[*sql.DB](db)
```

Named dependencies are also supported:

```go
depend.RegisterNamed[*sql.DB](db, "primary")
```

Variants that enforce uniqueness:

```go
depend.RegisterOnce[*sql.DB](db)
depend.RegisterNamedOnce[*sql.DB](db, "primary")
```

If a dependency is registered more than once when using the `Once` variants,
startup fails immediately.

### Resolving Dependencies

Dependencies can be resolved directly:

```go
db, err := depend.Resolve[*sql.DB]()
db, err := depend.ResolveNamed[*sql.DB]("primary")
```

More commonly, dependencies are injected into structs via tags:

```go
type Service struct {
	DB        *sql.DB `resolve:""`
	PrimaryDB *sql.DB `resolve:"primary"`
}
```

Resolution happens during wiring. If a dependency cannot be resolved, the
application does not start.

Dependency registration and resolution events participate in introspection
and visualization.

---

## Package `config`

The `config` package provides **type-safe configuration loading and binding**,
also implemented using Go generics.

Configuration values are treated as first-class inputs to the application and
can be injected into components in the same way as dependencies.

### Configuration Providers

Configuration values are supplied by implementations of `config.Provider`.

Providers are responsible for retrieving raw configuration values from a source,
such as:

- environment variables
- configuration files
- secret managers
- external configuration services

A global provider can be set during initialization:

```go
config.SetGlobalProvider(config.NewEnvVarProvider())
```

Providers can be replaced or composed as needed.

#### Reading Configuration Values

Configuration values can be retrieved directly:

```go
port, err := config.Get[int](ctx, "APP_PORT")
port := config.GetWithDefault[int](ctx, "APP_PORT", 8080)
```

Missing keys, parse failures, or validation errors cause startup to fail early.

#### Struct Binding

Configuration can also be loaded directly into structs using tags:

```go
type AppConfig struct {
	Port int    `config:"APP_PORT" default:"8080"`
	DSN  string `config:"DB_DSN"`
}

var cfg AppConfig
if err := config.LoadStruct(ctx, &cfg); err != nil {
	return err
}
```

This allows configuration to be validated and injected before any runtime
logic begins.

### Custom Parsers

Custom parsers can be registered for complex or domain-specific types:

```go
config.RegisterParser[[]string](func(v string) ([]string, error) {
	return strings.Split(v, ","), nil
})
```

Once registered, custom types can be used transparently:

```go
values, err := config.Get[[]string](ctx, "MY_LIST")
```

All configuration access is tracked by the introspection system and can be
visualized alongside dependencies.