package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/acai-travel/tech-challenge/internal/chat"
	"github.com/acai-travel/tech-challenge/internal/chat/assistant"
	"github.com/acai-travel/tech-challenge/internal/chat/model"
	"github.com/acai-travel/tech-challenge/internal/httpx"
	"github.com/acai-travel/tech-challenge/internal/mongox"
	"github.com/acai-travel/tech-challenge/internal/pb"
	"github.com/gorilla/mux"
	"github.com/twitchtv/twirp"
	"go.opentelemetry.io/otel"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup OpenTelemetry instrumentation
	cleanup, err := httpx.SetupOpenTelemetry(ctx, "assistant-service")
	if err != nil {
		slog.ErrorContext(ctx, "Failed to setup OpenTelemetry", "error", err)
		panic(err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
		defer shutdownCancel()
		if err := cleanup(shutdownCtx); err != nil {
			slog.ErrorContext(shutdownCtx, "Failed to cleanup OpenTelemetry", "error", err)
		}
	}()

	mongo := mongox.MustConnect()

	repo := model.New(mongo)
	assist := assistant.New()

	server := chat.NewServer(repo, assist)

	// Get meter for metrics middleware
	meter := otel.GetMeterProvider().Meter("internal/httpx")
	metricsMiddleware, err := httpx.NewMetricsMiddleware(meter)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create metrics middleware", "error", err)
		panic(err)
	}

	// Configure handler
	handler := mux.NewRouter()
	handler.Use(
		httpx.Logger(),
		httpx.Recovery(),
		metricsMiddleware,
		httpx.NewTracingMiddleware(),
	)

	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "Hi, my name is Clippy!")
	})

	handler.PathPrefix("/twirp/").Handler(pb.NewChatServiceServer(server, twirp.WithServerJSONSkipDefaults(true)))

	// Determine port from `PORT` env var (default 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	// Start server in a goroutine
	slog.InfoContext(ctx, "Starting the server", "addr", addr)
	go func() {
		if err := http.ListenAndServe(addr, handler); err != nil {
			slog.ErrorContext(ctx, "Server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.InfoContext(ctx, "Shutting down server...")
}
