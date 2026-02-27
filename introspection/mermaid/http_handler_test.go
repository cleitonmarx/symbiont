package mermaid

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/stretchr/testify/assert"
)

func TestNewGraphHandler(t *testing.T) {
	type tc struct {
		name     string
		appName  string
		report   introspection.Report
		opts     []GraphHandlerOption
		validate func(t *testing.T, rec *httptest.ResponseRecorder)
	}

	cases := []tc{
		{
			name:    "serves-html",
			appName: "MyApp",
			report:  introspection.Report{},
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				body := rec.Body.String()
				assert.Equal(t, http.StatusOK, rec.Code)
				assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
				assert.Contains(t, body, "<title>MyApp Introspection Graph</title>")
				assert.Contains(t, body, "mermaid.render('mermaid-svg-id',")
				assert.Regexp(t, regexp.MustCompile(`maxTextSize:\s*100000`), body)
			},
		},
		{
			name:    "embeds-graph-as-escaped-js-string",
			appName: "App<title>",
			report:  introspection.Report{},
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				body := rec.Body.String()
				assert.Equal(t, http.StatusOK, rec.Code)
				assert.Contains(t, body, "&lt;title&gt; Introspection Graph")
				assert.Contains(t, body, `mermaid.render('mermaid-svg-id', "---\n  config:\n    layout: elk\n---\ngraph TD\n`)
				assert.NotContains(t, body, "mermaid.render('mermaid-svg-id', \"---\n  config:\n")
				assert.Equal(t, 1, strings.Count(body, "mermaid.render('mermaid-svg-id',"))
			},
		},
		{
			name:    "overrides-max-text-size",
			appName: "MyApp",
			report:  introspection.Report{},
			opts:    []GraphHandlerOption{WithMaxTextSize(2048)},
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				assert.Regexp(t, regexp.MustCompile(`maxTextSize:\s*2048`), rec.Body.String())
			},
		},
		{
			name:    "invalid-max-text-size-uses-default",
			appName: "MyApp",
			report:  introspection.Report{},
			opts:    []GraphHandlerOption{WithMaxTextSize(0)},
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				assert.Regexp(t, regexp.MustCompile(`maxTextSize:\s*100000`), rec.Body.String())
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			handler := NewGraphHandler(c.appName, c.report, c.opts...)
			req := httptest.NewRequest(http.MethodGet, "/introspection", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)
			c.validate(t, rec)
		})
	}
}
