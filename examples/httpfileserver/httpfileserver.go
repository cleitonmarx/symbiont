package httpfileserver

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/introspection"
)

type myIntrospector struct{}

func (i *myIntrospector) Introspect(ctx context.Context, r introspection.Report) error {
	//fmt.Println(mermaid.GenerateIntrospectionGraph(r))
	return nil
}

type initLogger struct {
	Prefix string `config:"prefix" default:"httpFileServer->"`
}

func (i initLogger) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register(log.New(os.Stdout, i.Prefix, log.LstdFlags))
	return ctx, nil
}

type httpFileServer struct {
	Logger *log.Logger `resolve:""`
}

func (h httpFileServer) Run(ctx context.Context) error {
	h.Logger.Print("Starting")

	server := &http.Server{
		Addr:    ":12345",
		Handler: http.FileServer(http.Dir("./static")),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		h.Logger.Print("Shutting down")
		return server.Shutdown(ctx)
	case err := <-errCh:
		return err
	}
}

func (h httpFileServer) IsReady(ctx context.Context) error {
	resp, err := http.Get("http://:12345")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
