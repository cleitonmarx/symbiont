package tracing

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func TestSpanNameFormatter(t *testing.T) {
	req, _ := http.NewRequest("GET", "/foo/bar", nil)
	req.Pattern = "/foo/:bar"
	assert.Equal(t, "/foo/:bar", SpanNameFormatter("", req))

	req.Pattern = ""
	assert.Equal(t, "GET /foo/bar", SpanNameFormatter("", req))
}

func TestGetCallerName(t *testing.T) {
	name := getCallerName(0)
	assert.NotEmpty(t, name)
}

func TestRecordErrorAndStatus(t *testing.T) {
	span := &mockSpan{}
	err := errors.New("fail")
	assert.True(t, RecordErrorAndStatus(span, err))
	assert.Equal(t, "fail", span.lastError)
	assert.Equal(t, "fail", span.statusMsg)
	assert.Equal(t, codes.Error, span.statusCode) // codes.Error

	span = &mockSpan{}
	assert.False(t, RecordErrorAndStatus(span, nil))
	assert.Equal(t, "OK", span.statusMsg)
	assert.Equal(t, codes.Ok, span.statusCode) // codes.Ok
}

func TestInitOpenTelemetry_Initialize_Close(t *testing.T) {
	init := &InitOpenTelemetry{Logger: log.New(&strings.Builder{}, "", 0)}
	ctx := context.Background()
	ctx, err := init.Initialize(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
	init.Close()
}

func TestInitHttpClient_Initialize(t *testing.T) {
	init := InitHttpClient{Logger: log.New(&strings.Builder{}, "", 0)}
	ctx := context.Background()
	ctx, err := init.Initialize(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
}

// --- Mocks ---

type mockSpan struct {
	trace.Span
	lastError  string
	statusCode codes.Code
	statusMsg  string
}

func (m *mockSpan) RecordError(err error, _ ...trace.EventOption) {
	m.lastError = err.Error()
}
func (m *mockSpan) SetStatus(code codes.Code, msg string) {
	m.statusCode = code
	m.statusMsg = msg
}
