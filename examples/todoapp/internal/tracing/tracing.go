package tracing

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/hashicorp/go-retryablehttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer = otel.Tracer("")
)

// SpanNameFormatter formats span names for HTTP requests.
// It uses the HTTP method and URL path as the span name.
func SpanNameFormatter(_ string, r *http.Request) string {
	return getHttpRoute(r)
}

// WithHttpMetricAttributes returns attributes for HTTP metrics based on the request.
func WithHttpMetricAttributes(r *http.Request) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.HTTPRoute(getHttpRoute(r)),
	}
}

func getHttpRoute(r *http.Request) string {
	if r.Pattern != "" {
		return r.Pattern
	}
	return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
}

// Start a new span with the global tracer.
func Start(ctx context.Context, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tracer.Start(ctx, getCallerName(2), opts...)
}

// RecordErrorAndStatus records an error in the span and sets the status to Error.
// Returns true if an error was recorded, false otherwise.
func RecordErrorAndStatus(span trace.Span, err error) bool {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return true
	}
	span.SetStatus(codes.Ok, "OK")
	return false
}

// getCallerName retrieves the name of the function at the specified stack depth.
func getCallerName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}

	parts := strings.Split(fn.Name(), "/")

	return strings.ReplaceAll(parts[len(parts)-1], ".", "::")
}

// InitOpenTelemetry is a component that sets up OpenTelemetry tracing.
type InitOpenTelemetry struct {
	Logger *log.Logger `resolve:""`
	tp     *sdktrace.TracerProvider
	se     sdktrace.SpanExporter
	mp     *sdkmetric.MeterProvider
	me     sdkmetric.Exporter
}

// Initialize sets up OpenTelemetry tracing and exporting.
func (o *InitOpenTelemetry) Initialize(ctx context.Context) (context.Context, error) {
	var err error
	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up resource.
	res, err := newAppResource(ctx)
	if err != nil {
		return ctx, err
	}

	// Set up trace provider.
	o.tp, o.se, err = newTracerProvider(ctx, res)
	if err != nil {
		return ctx, err
	}
	otel.SetTracerProvider(o.tp)

	// Set up meter provider.
	o.mp, o.me, err = newMeterProvider(ctx, res)
	if err != nil {
		return ctx, err
	}
	otel.SetMeterProvider(o.mp)

	return ctx, nil
}

// Close shuts down the OpenTelemetry tracer provider and span exporter.
func (o *InitOpenTelemetry) Close() {
	cancelCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := o.tp.Shutdown(cancelCtx); err != nil {
		o.Logger.Printf("Error shutting down tracer provider: %v", err)
	}
	if err := o.se.Shutdown(cancelCtx); err != nil {
		o.Logger.Printf("Error shutting down span exporter: %v", err)
	}
	if err := o.mp.Shutdown(cancelCtx); err != nil {
		o.Logger.Printf("Error shutting down meter provider: %v", err)
	}
	if err := o.me.Shutdown(cancelCtx); err != nil {
		o.Logger.Printf("Error shutting down meter exporter: %v", err)
	}
}

// newPropagator creates a new composite text map propagator.
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newAppResource(ctx context.Context) (*resource.Resource, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("todoapp"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	return res, nil
}

// newTracerProvider creates a new tracer provider with an OTLP HTTP exporter.
func newTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, sdktrace.SpanExporter, error) {
	otlpExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(otlpExporter,
			sdktrace.WithBatchTimeout(time.Second),
		),
		sdktrace.WithResource(res),
	)
	return tracerProvider, otlpExporter, nil
}

func newMeterProvider(ctx context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, sdkmetric.Exporter, error) {
	exporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithInsecure())
	if err != nil {
		return nil, nil, err
	}

	// 2. Create the MeterProvider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
			exporter,
			sdkmetric.WithInterval(5*time.Second),
		)),
		// This view configures histogram aggregation for all duration instruments
		// to have specific bucket boundaries.
		// This is useful for capturing latency distributions.
		sdkmetric.WithView(sdkmetric.NewView(
			sdkmetric.Instrument{Name: "*duration*"},
			sdkmetric.Stream{
				Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
					Boundaries: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
				},
			},
		)),
	)
	otel.SetMeterProvider(meterProvider)

	return meterProvider, exporter, nil
}

// InitHttpClient initializes an HTTP client instrumented with OpenTelemetry
// and with retry capabilities.
type InitHttpClient struct {
	Logger *log.Logger `resolve:""`
}

func (i InitHttpClient) Initialize(ctx context.Context) (context.Context, error) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryWaitMax = 5 * time.Second
	retryClient.RetryMax = 5
	retryClient.Logger = i.Logger

	stdClient := retryClient.StandardClient()
	stdClient.Transport = otelhttp.NewTransport(
		stdClient.Transport,
		otelhttp.WithSpanNameFormatter(SpanNameFormatter),
	)

	depend.Register(stdClient)
	return ctx, nil
}
