package httpx

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer = otel.Tracer("internal/httpx")
	
	attrHTTPMethod = attribute.Key("http.method")
	attrHTTPPath   = attribute.Key("http.path")
	attrHTTPStatus = attribute.Key("http.status")
)

// InitializeMetrics sets up the global meter provider for instrumentation.
// This should be called once at application startup before using metrics.
func InitializeMetrics(ctx context.Context, meterProvider metric.MeterProvider) error {
	// Set the global meter provider so NewMetricsMiddleware can use it
	otel.SetMeterProvider(meterProvider)
	return nil
}

// InitializeTracing sets up the global trace provider for instrumentation.
// This should be called once at application startup before using tracing.
func InitializeTracing(ctx context.Context, traceProvider trace.TracerProvider) error {
	// Set the global trace provider so tracer can use it
	otel.SetTracerProvider(traceProvider)
	return nil
}

// newRequestAttributes builds a set of attributes for HTTP request metrics.
func newRequestAttributes(method, path string, statusCode int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attrHTTPMethod.String(method),
		attrHTTPPath.String(path),
		attrHTTPStatus.Int(statusCode),
	}
}
