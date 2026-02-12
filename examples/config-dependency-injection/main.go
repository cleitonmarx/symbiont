package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/cleitonmarx/symbiont"
	"github.com/cleitonmarx/symbiont/depend"
)

// AppMetadata is shared runtime metadata injected into runnables.
type AppMetadata struct {
	ServiceName string
	Environment string
}

type startupIDContextKey struct{}

// AppMetadataInitializer loads configuration, registers dependencies, and enriches context.
type AppMetadataInitializer struct {
	ServiceName string `config:"SERVICE_NAME" default:"todo-api"`
	Environment string `config:"ENVIRONMENT" default:"local"`
}

// Initialize registers logger and metadata dependencies and stores a startup ID in context.
func (i AppMetadataInitializer) Initialize(ctx context.Context) (context.Context, error) {
	logger := log.New(os.Stdout, "[config-di] ", log.LstdFlags)
	depend.Register(logger)
	depend.Register(AppMetadata(i))

	startupID := time.Now().UTC().Format(time.RFC3339Nano)
	ctx = context.WithValue(ctx, startupIDContextKey{}, startupID)

	return ctx, nil
}

// ConfiguredWorker demonstrates config and dependency injection in one runnable.
type ConfiguredWorker struct {
	Logger       *log.Logger   `resolve:""`
	Metadata     AppMetadata   `resolve:""`
	PollInterval time.Duration `config:"POLL_INTERVAL" default:"2s"`
}

// Run starts the worker, reads startup ID from context, and logs injected values until shutdown.
func (w ConfiguredWorker) Run(ctx context.Context) error {
	startupID, _ := ctx.Value(startupIDContextKey{}).(string)
	if startupID == "" {
		startupID = "unknown"
	}

	w.Logger.Printf("worker started service=%s env=%s poll_interval=%s startup_id=%s", w.Metadata.ServiceName, w.Metadata.Environment, w.PollInterval, startupID)

	ticker := time.NewTicker(w.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.Logger.Println("worker stopped")
			return nil
		case <-ticker.C:
			w.Logger.Printf("processing tick service=%s env=%s startup_id=%s", w.Metadata.ServiceName, w.Metadata.Environment, startupID)
		}
	}
}

func main() {
	app := symbiont.NewApp().
		Initialize(&AppMetadataInitializer{}).
		Host(&ConfiguredWorker{})

	if err := app.Run(); err != nil {
		log.Fatalf("app failed: %v", err)
	}
}
