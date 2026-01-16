package examples

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cleitonmarx/symbiont"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/davecgh/go-spew/spew"
)

type introspector struct{}

func (i introspector) Introspect(_ context.Context, ai symbiont.AppIntrospection) error {
	spew.Dump(ai)
	fmt.Println(ai.GenerateIntrospectionGraph())
	return nil
}

func Example_httpFileServer() {
	cancelCtx, cancel := context.WithCancel(context.Background())

	app := symbiont.NewApp().
		Initialize(&initLogger{}).
		Host(&httpFileServer{}).
		Instrospect(&introspector{})
	shutdownCh := app.RunAsync(cancelCtx)

	err := app.WaitForReadiness(cancelCtx, 10000*time.Second)
	if err != nil {
		log.Fatalf("Application failed to start: %v", err)
	}

	// Send a test HTTP request to verify the server is running
	sendHttpRequest()

	// Cancel the context to trigger a graceful shutdown
	cancel()

	select {
	case err = <-shutdownCh:
		if err != nil {
			fmt.Println("Application shutdown with error:", err)
		} else {
			fmt.Println("Application shutdown complete")
		}
	case <-time.After(1 * time.Second):
		fmt.Printf("Application shutdown timed out")
	}

	// Unordered Output:
	// httpFileServer: starting
	// Received response with status code: 200
	// Response body: <!DOCTYPE html>
	// <html>
	// <body>
	//     <h1>Hello, World!</h1>
	// </body>
	// </html>
	// httpFileServer: stopping
	// Application shutdown complete
}

type initLogger struct{}

func (i *initLogger) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register(log.New(os.Stdout, "httpFileServer: ", log.Lmsgprefix))
	return ctx, nil
}

type httpFileServer struct {
	Logger *log.Logger `resolve:""`
}

func (h *httpFileServer) Run(ctx context.Context) error {
	h.Logger.Print("starting")

	server := &http.Server{
		Addr:    ":8080",
		Handler: http.FileServer(http.Dir("./static")),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		h.Logger.Print("stopping")
		return server.Shutdown(ctx)
	case err := <-errCh:
		return err
	}
}

func (h *httpFileServer) IsReady(ctx context.Context) error {
	resp, err := http.Get("http://:8080")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func sendHttpRequest() {
	resp, err := http.Get("http://:8080")
	if err != nil {
		log.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body) // Read the body to completion
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}
	fmt.Println("Received response with status code:", resp.StatusCode)
	fmt.Println("Response body:", string(b))
}
