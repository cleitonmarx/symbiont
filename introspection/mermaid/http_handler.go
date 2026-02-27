package mermaid

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/cleitonmarx/symbiont/introspection"
)

var (
	//go:embed introspect.gohtml assets
	templateFS embed.FS
	tmpl       = template.Must(template.ParseFS(templateFS, "introspect.gohtml"))
)

// defaultMaxTextSize is the default value for Mermaid's maxTextSize,
// which limits the size of text in nodes and edges.
// See https://mermaid.js.org/config/schema-docs/config-properties-maxtextsize.html#maxtextsize-type for details.
const defaultMaxTextSize = 100000

// graphHandlerConfig holds configuration options for the graph HTTP handler.
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

// graphPageData holds the data passed to the HTML template for rendering the graph page.
type graphPageData struct {
	Graph       string
	Title       string
	MaxTextSize int
}

// NewGraphHandler creates an HTTP handler that serves an introspection graph page
// and its local JavaScript assets from the same origin.
func NewGraphHandler(appName string, report introspection.Report, opts ...GraphHandlerOption) http.Handler {
	cfg := graphHandlerConfig{
		maxTextSize: defaultMaxTextSize,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	graph := GenerateIntrospectionGraph(report)

	var out bytes.Buffer
	if err := tmpl.Execute(&out, graphPageData{
		Title:       fmt.Sprintf("%s Introspection Graph", appName),
		MaxTextSize: cfg.maxTextSize,
		Graph:       graph,
	}); err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		})
	}

	page := append([]byte(nil), out.Bytes()...)
	assetsFS := mustSubFS(templateFS, "assets")
	_ = mime.AddExtensionType(".mjs", "text/javascript; charset=utf-8")
	assetHandler := http.FileServer(http.FS(assetsFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if assetPath, ok := assetRequestPath(r.URL.Path); ok {
			req := r.Clone(r.Context())
			req.URL.Path = "/" + assetPath
			assetHandler.ServeHTTP(w, req)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	})
}

// mustSubFS is a helper function that returns a subdirectory of an fs.FS or panics if it fails.
func mustSubFS(root fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(root, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

// assetRequestPath checks if the request path is for an asset and returns the cleaned asset path if so.
func assetRequestPath(requestPath string) (string, bool) {
	_, after, ok := strings.Cut(requestPath, "/assets/")
	if !ok {
		return "", false
	}

	assetPath := path.Clean(after)
	if assetPath == "." || assetPath == "" || strings.HasPrefix(assetPath, "../") {
		return "", false
	}

	return assetPath, true
}
