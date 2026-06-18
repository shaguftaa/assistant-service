package tools

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegistryRunUnknownTool(t *testing.T) {
	reg := Default()

	_, err := reg.Run(context.Background(), "missing_tool", `{}`)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}

	if !errors.Is(err, ErrUnknownTool) {
		t.Fatalf("expected ErrUnknownTool, got %v", err)
	}
}

func TestExchangeRateToolRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"amount": 10.0,
			"base":   "USD",
			"date":   "2026-06-17",
			"rates": map[string]float64{
				"EUR": 9.2,
			},
		})
	}))
	defer server.Close()

	tool := &ExchangeRateTool{
		http:    server.Client(),
		baseURL: server.URL,
	}

	result, err := tool.Run(context.Background(), `{"from":"USD","to":"EUR","amount":10}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == "" {
		t.Fatal("expected non-empty result")
	}
}
