# Examples

This directory contains minimal runnable examples for core Symbiont patterns.

In `config-dependency-injection`, the initializer also enriches `context.Context`
with a `startup_id` value that the worker reads during `Run`.

## Run

```shell
go run ./examples/single-hosting
go run ./examples/multiple-hosting
go run ./examples/config-dependency-injection
```

To try custom configuration values in the config example:

```shell
SERVICE_NAME=todo-api ENVIRONMENT=dev POLL_INTERVAL=1s go run ./examples/config-dependency-injection
```

Press `Ctrl+C` to stop the examples.
