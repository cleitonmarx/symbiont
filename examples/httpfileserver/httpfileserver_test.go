package httpfileserver

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_httpFileServer(t *testing.T) {
	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup the application
	app := symbiont.NewApp().
		Initialize(&initLogger{}).
		Host(&httpFileServer{}).
		Instrospect(&myIntrospector{})

	// Run the application asynchronously
	shutdownCh := app.RunAsync(cancelCtx)

	// Wait for the application to become ready
	err := app.WaitForReadiness(cancelCtx, 4*time.Second)
	if err != nil {
		log.Fatalf("Application failed to start: %v", err)
	}

	// Make a request to the HTTP file server
	resp, err := http.Get("http://:12345")
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	// Read the expected file content
	file, err := os.OpenFile("./static/index.html", os.O_RDONLY, 0644)
	assert.NoError(t, err)
	defer file.Close()

	//Assert the response body matches the expected file content
	expectedBody, err := io.ReadAll(file)
	assert.NoError(t, err)
	assert.Equal(t, string(expectedBody), string(body))

	// Cancel the context to trigger a graceful shutdown
	cancel()

	// Wait for the application to shutdown
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
