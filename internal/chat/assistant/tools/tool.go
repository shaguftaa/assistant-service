package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/openai/openai-go/v2"
)

var ErrUnknownTool = errors.New("unknown tool")

// Tool defines a callable assistant capability exposed to the model.
type Tool interface {
	Name() string
	Definition() openai.ChatCompletionToolUnionParam
	Run(ctx context.Context, arguments string) (string, error)
}

// Registry holds tools by name and exposes OpenAI definitions plus dispatch.
type Registry struct {
	byName map[string]Tool
	order  []Tool
}

// NewRegistry builds a registry from the provided tools.
func NewRegistry(tools ...Tool) *Registry {
	r := &Registry{
		byName: make(map[string]Tool, len(tools)),
		order:  make([]Tool, 0, len(tools)),
	}

	for _, tool := range tools {
		r.Register(tool)
	}

	return r
}

// Default returns the standard tool set used by the assistant.
func Default() *Registry {
	return NewRegistry(
		NewWeatherTool(),
		NewTodayDateTool(),
		NewHolidaysTool(),
		NewExchangeRateTool(),
	)
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool Tool) {
	name := tool.Name()
	if _, exists := r.byName[name]; exists {
		panic(fmt.Sprintf("tool %q already registered", name))
	}

	r.byName[name] = tool
	r.order = append(r.order, tool)
}

// Definitions returns OpenAI tool schemas in registration order.
func (r *Registry) Definitions() []openai.ChatCompletionToolUnionParam {
	defs := make([]openai.ChatCompletionToolUnionParam, len(r.order))
	for i, tool := range r.order {
		defs[i] = tool.Definition()
	}
	return defs
}

// Run executes the named tool with JSON arguments from the model.
func (r *Registry) Run(ctx context.Context, name, arguments string) (string, error) {
	tool, ok := r.byName[name]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnknownTool, name)
	}

	return tool.Run(ctx, arguments)
}

func functionTool(name, description string, parameters openai.FunctionParameters) openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        name,
		Description: openai.String(description),
		Parameters:  parameters,
	})
}
