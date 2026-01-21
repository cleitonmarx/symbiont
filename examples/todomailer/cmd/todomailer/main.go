package main

import (
	"context"
	glog "log"

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

func main() {
	// This main.go file is intentionally left blank.
	// The actual server is started in internal/adapters/inbound/http/todomailerapi.go
	err := symbiont.NewApp().
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
		).
		Host(
			&http.TodoMailerApp{},
			&worker.TodoEmailSender{},
		).
		Instrospect(&myIntrospector{}).
		Run()
	if err != nil {
		panic(err)
	}
}

// myIntrospector is an implementation of introspection.Introspector that logs the introspection report.
type myIntrospector struct {
	Logger *glog.Logger `resolve:""`
}

// Introspect generates and logs the introspection report and a Mermaid graph.
func (i *myIntrospector) Introspect(ctx context.Context, r introspection.Report) error {
	b, err := r.ToJSON()
	if err != nil {
		return err
	}
	i.Logger.Println("=== TODOMAILER INTROSPECTION REPORT ===")
	i.Logger.Println(string(b))
	i.Logger.Println(mermaid.GenerateIntrospectionGraph(r))
	i.Logger.Println("=== END OF REPORT ===")
	return nil
}
