package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/cleitonmarx/symbiont"
	"github.com/cleitonmarx/symbiont/depend"
)

// LoggerInitializer registers the logger dependency for all runnables.
type LoggerInitializer struct{}

// Initialize registers logger dependency for hosted runnables.
func (LoggerInitializer) Initialize(ctx context.Context) (context.Context, error) {
	logger := log.New(os.Stdout, "[multi] ", log.LstdFlags)
	depend.Register(logger)
	return ctx, nil
}

// HTTPWorker simulates an HTTP service runnable.
type HTTPWorker struct {
	Logger *log.Logger `resolve:""`
}

// Run starts the simulated HTTP service until shutdown.
func (w HTTPWorker) Run(ctx context.Context) error {
	w.Logger.Println("http worker started")

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.Logger.Println("http worker stopped")
			return nil
		case <-ticker.C:
			w.Logger.Println("http worker: handled request")
		}
	}
}

// MetricsWorker simulates a background metrics runnable.
type MetricsWorker struct {
	Logger *log.Logger `resolve:""`
}

// Run starts metrics collection until shutdown.
func (w MetricsWorker) Run(ctx context.Context) error {
	w.Logger.Println("metrics worker started")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.Logger.Println("metrics worker stopped")
			return nil
		case <-ticker.C:
			w.Logger.Println("metrics worker: flushed counters")
		}
	}
}

func main() {
	app := symbiont.NewApp().
		Initialize(&LoggerInitializer{}).
		Host(
			&HTTPWorker{},
			&MetricsWorker{},
		)

	if err := app.Run(); err != nil {
		log.Fatalf("app failed: %v", err)
	}
}
