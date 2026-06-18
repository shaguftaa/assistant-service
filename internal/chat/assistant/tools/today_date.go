package tools

import (
	"context"
	"time"

	"github.com/openai/openai-go/v2"
)

type TodayDateTool struct{}

func NewTodayDateTool() *TodayDateTool {
	return &TodayDateTool{}
}

func (t *TodayDateTool) Name() string {
	return "get_today_date"
}

func (t *TodayDateTool) Definition() openai.ChatCompletionToolUnionParam {
	return functionTool(t.Name(), "Get today's date and time in RFC3339 format", nil)
}

func (t *TodayDateTool) Run(ctx context.Context, arguments string) (string, error) {
	return time.Now().Format(time.RFC3339), nil
}
