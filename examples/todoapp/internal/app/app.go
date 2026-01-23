package app

import (
	"context"
	stdlog "log"

	"github.com/cleitonmarx/symbiont"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/worker"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/outbound/config"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/outbound/llm"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/outbound/log"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/outbound/postgres"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/outbound/pubsub"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/outbound/time"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/cleitonmarx/symbiont/introspection/mermaid"
)

// NewTodoApp creates and returns a new instance of the TodoApp application.
func NewTodoApp(initializers ...symbiont.Initializer) *symbiont.App {
	return symbiont.NewApp().
		Initialize(initializers...).
		Initialize(
			&log.InitLogger{},
			&tracing.InitOpenTelemetry{},
			&tracing.InitHttpClient{},
			&config.InitVaultProvider{},
			&postgres.InitDB{},
			&postgres.InitUnitOfWork{},
			&postgres.InitTodoRepository{},
			&postgres.InitBoardSummaryRepository{},
			&time.InitCurrentTimeProvider{},
			&pubsub.InitClient{},
			&llm.InitBoardSummaryGenerator{},
			&usecases.InitListTodos{},
			&usecases.InitCreateTodo{},
			&usecases.InitUpdateTodo{},
			&usecases.InitGenerateBoardSummary{},
			&usecases.InitGetBoardSummary{},
		).
		Host(
			&http.TodoAppServer{},
			&worker.TodoEventSubscriber{},
			&worker.OutboxPublisher{},
		)
}

// ReportLoggerIntrospector is an implementation of introspection.Introspector that logs the introspection report.
type ReportLoggerIntrospector struct {
	Logger *stdlog.Logger `resolve:""`
}

// Introspect generates and logs the introspection report and a Mermaid graph.
func (i ReportLoggerIntrospector) Introspect(ctx context.Context, r introspection.Report) error {
	b, err := r.ToJSON()
	if err != nil {
		return err
	}
	i.Logger.Println("=== TODOAPP INTROSPECTION REPORT ===")
	i.Logger.Println(string(b))
	i.Logger.Println("=== MERMAID GRAPH ===")
	i.Logger.Println(mermaid.GenerateIntrospectionGraph(r))
	i.Logger.Println("=== END OF REPORT ===")
	return nil
}
