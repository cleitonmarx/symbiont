package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/cleitonmarx/symbiont"
	"github.com/cleitonmarx/symbiont/depend"
)

// LoggerInitializer registers a shared logger dependency.
type LoggerInitializer struct{}

// Initialize registers logger dependency for hosted runnables.
func (LoggerInitializer) Initialize(ctx context.Context) (context.Context, error) {
	logger := log.New(os.Stdout, "[single] ", log.LstdFlags)
	depend.Register(logger)
	return ctx, nil
}

// SingleWorker is a single hosted runnable.
type SingleWorker struct {
	Logger *log.Logger `resolve:""`
}

// Run starts a loop until the app context is canceled.
func (w SingleWorker) Run(ctx context.Context) error {
	w.Logger.Println("worker started")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.Logger.Println("worker stopped")
			return nil
		case now := <-ticker.C:
			w.Logger.Printf("tick: %s", now.Format(time.RFC3339))
		}
	}
}

func main() {
	app := symbiont.NewApp().
		Initialize(&LoggerInitializer{}).
		Host(&SingleWorker{})

	if err := app.Run(); err != nil {
		log.Fatalf("app failed: %v", err)
	}
}
