# OpenTelemetry Instrumentation

This document describes the OpenTelemetry instrumentation added to the web server for tracking metrics and traces.

## Overview

The assistant-service now includes built-in instrumentation for monitoring request performance and tracing request flow. The implementation uses OpenTelemetry with stdout exporters for simple visibility during development.

## Metrics

### HTTP Request Count
- **Metric Name:** `http.server.request.count`
- **Unit:** requests (1)
- **Type:** Counter (incremented for each request)
- **Attributes:**
  - `http.method`: HTTP method (GET, POST, etc.)
  - `http.path`: Request path
  - `http.status`: HTTP status code

### HTTP Request Duration
- **Metric Name:** `http.server.request.duration`
- **Unit:** seconds (s)
- **Type:** Histogram (records latency distribution)
- **Attributes:**
  - `http.method`: HTTP method
  - `http.path`: Request path
  - `http.status`: HTTP status code

## Tracing

Each HTTP request creates a span with:
- **Span Name:** `{METHOD} {PATH}` (e.g., "GET /twirp/acai.chat.ChatService/DescribeConversation")
- **Attributes:**
  - `http.method`: Request method
  - `http.path`: Request path
  - `http.status`: Response status code

Traces are automatically propagated through the request context, allowing downstream operations to create child spans.

## Configuration

### Setup
The instrumentation is initialized in `cmd/server/main.go`:

```go
cleanup, err := httpx.SetupOpenTelemetry(ctx, "assistant-service")
if err != nil {
    panic(err)
}
defer cleanup(shutdownCtx)
```

### Middleware Registration
Metrics and tracing middleware are registered as Gorilla mux middleware:

```go
metricsMiddleware, err := httpx.NewMetricsMiddleware(meter)
handler.Use(
    httpx.Logger(),
    httpx.Recovery(),
    metricsMiddleware,
    httpx.NewTracingMiddleware(),
)
```

## Output Format

### Metrics Output (JSON)
```json
{
  "ResourceMetrics": [
        "Attributes": [
          {
            "Key": "service.name",
            "Value": {
              "StringValue": "assistant-service"
            }
        ]
      },
      "ScopeMetrics": [
        {
          "Metrics": [
            {
              "Name": "http.server.request.count",
              "Sum": {
                "DataPoints": [
                  {
                    "Attributes": [
                      {"Key": "http.method", "Value": {"StringValue": "GET"}},
                      {"Key": "http.path", "Value": {"StringValue": "/"}},
                      {"Key": "http.status", "Value": {"IntValue": "200"}}
                    ],
                    "Value": 1
                  }
                ]
              }
            }
          ]
        }
      ]
    }
  ]
}
```

### Traces Output (JSON)
```json
{
  "ResourceSpans": [
    {
      "Resource": {
        "Attributes": [
          {
            "Key": "service.name",
            "Value": {"StringValue": "assistant-service"}
          }
        ]
      },
      "ScopeSpans": [
        {
          "Spans": [
            {
              "Name": "GET /",
              "Status": {"Code": 0},
              "Attributes": [
                {"Key": "http.method", "Value": {"StringValue": "GET"}},
                {"Key": "http.path", "Value": {"StringValue": "/"}},
                {"Key": "http.status", "Value": {"IntValue": "200"}}
              ]
            }
          ]
        }
      ]
    }
  ]
}
```


## Testing

Instrumentation includes unit tests in `internal/httpx/metrics_test.go`:
- `TestMetricsMiddleware`: Verifies metrics capture
- `TestTracingMiddleware`: Verifies tracing functionality

Run tests with:
```bash
go test -count=1 ./internal/httpx/
```

---

## Quick Start: Build & Test Metrics/Traces

To test the metrics and tracing instrumentation locally:

### 1. Build the server binary

```bash
go build -o server ./cmd/server
```

### 2. Set environment variables

```bash
export OPENAI_API_KEY=your_openai_api_key
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4318
```

### 3. Start MongoDB (if testing with the full service)

```bash
make up
```

### 4. Run the server

```bash
./server
```

You should see metrics and trace output in the console (stdout).

### 5. Generate traffic

In another terminal:

```bash
curl http://localhost:8080/
curl -X POST http://localhost:8080/twirp/acai.chat.ChatService/StartConversation -d '{"message":"Hi"}'
```

Watch the server terminal to see request metrics and traces logged.

---

## Jaeger (local) — view traces in Jaeger UI

The service sends traces over OTLP/HTTP by default. This section explains how to run Jaeger locally and view traces for `assistant-service`.

1. Start Jaeger all-in-one (includes UI and OTLP/HTTP receiver):

```bash
docker run -d --name jaeger \
  -e COLLECTOR_OTLP_ENABLED=true \
  -p 16686:16686 \
  -p 4318:4318 \
  docker.io/jaegertracing/all-in-one:1.49
```

2. Start the service (OTLP endpoint defaults to `localhost:4318`):

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4318 go run ./cmd/server
```

3. Generate traffic (examples):

```bash
curl http://localhost:8080/
curl -X POST http://localhost:8080/twirp/acai.chat.ChatService/StartConversation -d '{"message":"Hi"}'
```

4. Open Jaeger UI at: http://localhost:16686 and select service `assistant-service` to inspect traces.

Notes:
- Configure exporter endpoint: `internal/httpx/setup.go` (uses `OTEL_EXPORTER_OTLP_ENDPOINT`).
- Middleware files: `internal/httpx/metrics.go` and `internal/httpx/instrumentation.go`.
- Middleware registration: `cmd/server/main.go`.

---

## References

- [OpenTelemetry Go Documentation](https://opentelemetry.io/docs/languages/go/)
- [OpenTelemetry Metrics](https://opentelemetry.io/docs/specs/otel/metrics/)
- [OpenTelemetry Tracing](https://opentelemetry.io/docs/specs/otel/trace/)
