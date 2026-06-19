package httpx

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel/metric"
)

type metricsMiddlewareState struct {
	requestCounter metric.Int64Counter
	requestLatency metric.Float64Histogram
}

// NewMetricsMiddleware creates a middleware that captures request metrics.
// It tracks:
//   - http.server.request.count: Number of HTTP requests
//   - http.server.request.duration: Time taken to process requests (in seconds)
func NewMetricsMiddleware(meter metric.Meter) (func(http.Handler) http.Handler, error) {
	requestCounter, err := meter.Int64Counter(
		"http.server.request.count",
		metric.WithDescription("Number of HTTP requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	requestLatency, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("HTTP request latency"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	state := &metricsMiddlewareState{
		requestCounter: requestCounter,
		requestLatency: requestLatency,
	}

	return state.middleware, nil
}

func (s *metricsMiddlewareState) middleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		saw := &statusAwareResponseWriter{ResponseWriter: w}

		defer func() {
			duration := time.Since(start).Seconds()

			// Record metrics with attributes
			attrs := newRequestAttributes(r.Method, r.URL.Path, saw.status)

			s.requestCounter.Add(r.Context(), 1, metric.WithAttributes(attrs...))
			s.requestLatency.Record(r.Context(), duration, metric.WithAttributes(attrs...))
		}()

		handler.ServeHTTP(saw, r)
	})
}

// NewTracingMiddleware creates a middleware that traces request flow.
// It starts a span for each HTTP request and records method, path, and status code.
func NewTracingMiddleware() func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
			defer span.End()

			saw := &statusAwareResponseWriter{ResponseWriter: w}

			defer func() {
				if saw.status > 0 {
					span.SetAttributes(
						attrHTTPMethod.String(r.Method),
						attrHTTPPath.String(r.URL.Path),
						attrHTTPStatus.Int(saw.status),
					)
				}
			}()

			handler.ServeHTTP(saw, r.WithContext(ctx))
		})
	}
}
