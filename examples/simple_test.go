package examples

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/cleitonmarx/symbiont"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/cleitonmarx/symbiont/introspection/mermaid"
	"github.com/davecgh/go-spew/spew"
)

type introspector struct{}

func (i introspector) Introspect(_ context.Context, r introspection.Report) error {
	spew.Dump(r)
	fmt.Println(mermaid.GenerateIntrospectionGraph(r))
	return nil
}

func Example_httpFileServer() {
	cancelCtx, cancel := context.WithCancel(context.Background())
	addr, err := getAvailableAddress()
	if err != nil {
		fmt.Println("skipping example:", err)
		cancel()
		return
	}

	app := symbiont.NewApp().
		Initialize(&initLogger{}).
		Host(&httpFileServer{Addr: addr})
		//Instrospect(&introspector{})
	shutdownCh := app.RunAsync(cancelCtx)

	err = app.WaitForReadiness(cancelCtx, 10000*time.Second)
	if err != nil {
		log.Fatalf("Application failed to start: %v", err)
	}

	// Send a test HTTP request to verify the server is running
	sendHttpRequest(addr)

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

}

type initLogger struct{}

func (i *initLogger) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register(log.New(os.Stdout, "httpFileServer: ", log.Lmsgprefix))
	return ctx, nil
}

type httpFileServer struct {
	Logger *log.Logger `resolve:""`
	Addr   string
}

func (h *httpFileServer) Run(ctx context.Context) error {
	h.Logger.Print("starting")

	server := &http.Server{
		Addr:    h.Addr,
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
	resp, err := http.Get("http://" + h.Addr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func sendHttpRequest(addr string) {
	resp, err := http.Get("http://" + addr)
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

func getAvailableAddress() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to acquire test port: %w", err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr, nil
}
