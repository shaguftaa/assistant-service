package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/openai/openai-go/v2"
)

const exchangeRateAPIBase = "https://api.frankfurter.app/latest"

type ExchangeRateTool struct {
	http    *http.Client
	baseURL string
}

func NewExchangeRateTool() *ExchangeRateTool {
	return &ExchangeRateTool{
		http:    http.DefaultClient,
		baseURL: exchangeRateAPIBase,
	}
}

func (t *ExchangeRateTool) Name() string {
	return "get_exchange_rate"
}

func (t *ExchangeRateTool) Definition() openai.ChatCompletionToolUnionParam {
	return functionTool(t.Name(),
		"Get the latest exchange rate between two currencies. Useful for travel budgeting and price conversions.",
		openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"from": map[string]string{
					"type":        "string",
					"description": "Source currency code, e.g. USD or EUR",
				},
				"to": map[string]string{
					"type":        "string",
					"description": "Target currency code, e.g. INR or GBP",
				},
				"amount": map[string]string{
					"type":        "number",
					"description": "Optional amount to convert in the source currency. Defaults to 1.",
				},
			},
			"required": []string{"from", "to"},
		},
	)
}

func (t *ExchangeRateTool) Run(ctx context.Context, arguments string) (string, error) {
	var payload struct {
		From   string  `json:"from"`
		To     string  `json:"to"`
		Amount float64 `json:"amount"`
	}

	if err := json.Unmarshal([]byte(arguments), &payload); err != nil {
		return "", fmt.Errorf("parse arguments: %w", err)
	}

	from := strings.ToUpper(strings.TrimSpace(payload.From))
	to := strings.ToUpper(strings.TrimSpace(payload.To))
	if from == "" || to == "" {
		return "", fmt.Errorf("from and to currency codes are required")
	}

	amount := payload.Amount
	if amount <= 0 {
		amount = 1
	}

	endpoint := t.baseURL + "?" + url.Values{
		"from":   {from},
		"to":     {to},
		"amount": {fmt.Sprintf("%.2f", amount)},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("exchange rate API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read exchange rate API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("exchange rate API returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var result struct {
		Amount float64            `json:"amount"`
		Base   string             `json:"base"`
		Date   string             `json:"date"`
		Rates  map[string]float64 `json:"rates"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse exchange rate API response: %w", err)
	}

	rate, ok := result.Rates[to]
	if !ok {
		return "", fmt.Errorf("no rate returned for %s to %s", from, to)
	}

	return fmt.Sprintf(
		"Exchange rate on %s: %.2f %s = %.2f %s (rate %.4f)",
		result.Date,
		result.Amount,
		from,
		rate,
		to,
		rate/result.Amount,
	), nil
}
