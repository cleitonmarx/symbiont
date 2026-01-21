package app

import (
	"context"
	stdlog "log"

	"github.com/cleitonmarx/symbiont"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/adapters/inbound/http"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/adapters/inbound/worker"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/adapters/outbound/config"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/adapters/outbound/email"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/adapters/outbound/log"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/adapters/outbound/postgres"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/adapters/outbound/time"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/tracing"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/usecases"
	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/cleitonmarx/symbiont/introspection/mermaid"
)

// NewTodoMailerApp creates and returns a new instance of the TodoMailer application.
func NewTodoMailerApp(initializers ...symbiont.Initializer) *symbiont.App {
	return symbiont.NewApp().
		Initialize(initializers...).
		Initialize(
			&log.InitLogger{},
			&tracing.InitOpenTelemetry{},
			&config.InitVaultProvider{},
			&postgres.InitDB{},
			&postgres.InitTodoRepository{},
			&time.InitTimeService{},
			&email.InitEmailSender{},
			&usecases.InitListTodos{},
			&usecases.InitCreateTodo{},
			&usecases.InitUpdateTodo{},
			&usecases.InitSendDoneTodoEmails{},
			&usecases.InitGenerateBoardSummary{},
			&usecases.InitGetBoardSummary{},
		).
		Host(
			&http.TodoMailerApp{},
			&worker.TodoEmailSender{},
		)
}

// ReportLoggerIntrospector is an implementation of introspection.Introspector that logs the introspection report.
type ReportLoggerIntrospector struct {
	Logger *stdlog.Logger `resolve:""`
}

// Introspect generates and logs the introspection report and a Mermaid graph.
func (i *ReportLoggerIntrospector) Introspect(ctx context.Context, r introspection.Report) error {
	b, err := r.ToJSON()
	if err != nil {
		return err
	}
	i.Logger.Println("=== TODOMAILER INTROSPECTION REPORT ===")
	i.Logger.Println(string(b))
	i.Logger.Println("=== MERMAID GRAPH ===")
	i.Logger.Println(mermaid.GenerateIntrospectionGraph(r))
	i.Logger.Println("=== END OF REPORT ===")
	return nil
}
