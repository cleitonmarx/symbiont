package mermaid

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/cleitonmarx/symbiont/introspection"
)

var (
	//go:embed introspect.gohtml
	templateFS embed.FS
	tmpl       = template.Must(template.ParseFS(templateFS, "introspect.gohtml"))
)

const defaultMaxTextSize = 100000

type graphHandlerConfig struct {
	maxTextSize int
}

// GraphHandlerOption configures NewGraphHandler behavior.
type GraphHandlerOption func(*graphHandlerConfig)

// WithMaxTextSize sets Mermaid's maxTextSize value used by the graph page.
// Values <= 0 are ignored and default to 100000.
func WithMaxTextSize(maxTextSize int) GraphHandlerOption {
	return func(cfg *graphHandlerConfig) {
		if maxTextSize > 0 {
			cfg.maxTextSize = maxTextSize
		}
	}
}

type graphPageData struct {
	GraphJSON   template.JS
	Title       string
	MaxTextSize int
}

// NewGraphHandler creates an HTTP handler that serves an introspection graph of the application's configuration and dependencies.
func NewGraphHandler(appName string, report introspection.Report, opts ...GraphHandlerOption) http.Handler {
	cfg := graphHandlerConfig{
		maxTextSize: defaultMaxTextSize,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	graphJSON, err := json.Marshal(GenerateIntrospectionGraph(report))
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		})
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, graphPageData{
		Title:       fmt.Sprintf("%s Introspection Graph", appName),
		MaxTextSize: cfg.maxTextSize,
		// json.Marshal returns a valid JavaScript string literal for the graph source.
		GraphJSON: template.JS(string(graphJSON)),
	}); err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		})
	}
	page := append([]byte(nil), out.Bytes()...)

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	})
}
