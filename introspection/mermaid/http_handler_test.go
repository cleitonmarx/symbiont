package mermaid

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont/introspection"
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
				if rec.Code != http.StatusOK {
					t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
				}
				if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
					t.Fatalf("unexpected content type: %q", rec.Header().Get("Content-Type"))
				}
				if !strings.Contains(body, "<title>MyApp Introspection Graph</title>") {
					t.Fatalf("expected title in body")
				}
				if !strings.Contains(body, "mermaid.render('mermaid-svg-id',") {
					t.Fatalf("expected mermaid render call in body")
				}
				if !regexp.MustCompile(`maxTextSize:\s*100000`).MatchString(body) {
					t.Fatalf("expected default maxTextSize in body")
				}
			},
		},
		{
			name:    "embeds-graph-as-escaped-js-string",
			appName: "App<title>",
			report:  introspection.Report{},
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				body := rec.Body.String()
				if rec.Code != http.StatusOK {
					t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
				}
				if !strings.Contains(body, "&lt;title&gt; Introspection Graph") {
					t.Fatalf("expected escaped title in body")
				}
				if !strings.Contains(body, `mermaid.render('mermaid-svg-id', "---\n  config:\n    layout: elk\n---\ngraph TD\n`) {
					t.Fatalf("expected escaped graph string in body")
				}
				if strings.Contains(body, "mermaid.render('mermaid-svg-id', \"---\n  config:\n") {
					t.Fatalf("unexpected raw multiline graph string in body")
				}
				if strings.Count(body, "mermaid.render('mermaid-svg-id',") != 1 {
					t.Fatalf("expected one mermaid render call")
				}
			},
		},
		{
			name:    "overrides-max-text-size",
			appName: "MyApp",
			report:  introspection.Report{},
			opts:    []GraphHandlerOption{WithMaxTextSize(2048)},
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				if rec.Code != http.StatusOK {
					t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
				}
				if !regexp.MustCompile(`maxTextSize:\s*2048`).MatchString(rec.Body.String()) {
					t.Fatalf("expected custom maxTextSize in body")
				}
			},
		},
		{
			name:    "invalid-max-text-size-uses-default",
			appName: "MyApp",
			report:  introspection.Report{},
			opts:    []GraphHandlerOption{WithMaxTextSize(0)},
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				if rec.Code != http.StatusOK {
					t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
				}
				if !regexp.MustCompile(`maxTextSize:\s*100000`).MatchString(rec.Body.String()) {
					t.Fatalf("expected default maxTextSize in body")
				}
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
