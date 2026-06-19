package httpx

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	otlptracehttp "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.24.0"
)

// SetupOpenTelemetry initializes OpenTelemetry with a metrics exporter and an OTLP/HTTP
// traces exporter. Traces can be sent to a local Jaeger all-in-one instance via OTLP.
// The OTLP endpoint can be set via `OTEL_EXPORTER_OTLP_ENDPOINT` (host:port).
// Returns a cleanup function to shut down providers.
func SetupOpenTelemetry(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	// Setup metrics exporter (stdout for development)
	metricExporter, err := stdoutmetric.New()
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter)),
	)
	otel.SetMeterProvider(meterProvider)

	// Setup OTLP/HTTP trace exporter so traces can be ingested by Jaeger (or any OTLP
	// collector). The endpoint can be set via `OTEL_EXPORTER_OTLP_ENDPOINT` (host:port),
	// defaulting to localhost:4318 which is the usual OTLP/HTTP port for Jaeger all-in-one.
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otlpEndpoint == "" {
		otlpEndpoint = "localhost:4318"
	}

	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(otlpEndpoint),
		otlptracehttp.WithURLPath("/v1/traces"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(traceExporter),
	)
	otel.SetTracerProvider(traceProvider)

	// Return cleanup function
	cleanup := func(ctx context.Context) error {
		if err := meterProvider.Shutdown(ctx); err != nil {
			return err
		}
		return traceProvider.Shutdown(ctx)
	}

	return cleanup, nil
}
