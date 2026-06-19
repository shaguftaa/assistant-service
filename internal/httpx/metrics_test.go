package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv/v1.24.0"
)

func TestMetricsMiddleware(t *testing.T) {
	ctx := context.Background()

	// Setup a test meter provider
	_, err := stdoutmetric.New()
	if err != nil {
		t.Fatalf("failed to create exporter: %v", err)
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceNameKey.String("test"),
	))
	if err != nil {
		t.Fatalf("failed to create resource: %v", err)
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewManualReader()),
	)
	otel.SetMeterProvider(meterProvider)

	meter := otel.GetMeterProvider().Meter("test")
	metricsMiddleware, err := NewMetricsMiddleware(meter)
	if err != nil {
		t.Fatalf("failed to create metrics middleware: %v", err)
	}

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap with middleware
	wrappedHandler := metricsMiddleware(testHandler)

	// Make a test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Fatalf("expected body 'OK', got %q", w.Body.String())
	}
}

func TestTracingMiddleware(t *testing.T) {
	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("traced"))
	})

	tracingMiddleware := NewTracingMiddleware()
	wrappedHandler := tracingMiddleware(testHandler)

	// Make a test request
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "traced" {
		t.Fatalf("expected body 'traced', got %q", w.Body.String())
	}
}
