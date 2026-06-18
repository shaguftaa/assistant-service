package assistant

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/acai-travel/tech-challenge/internal/chat/assistant/tools"
	"github.com/acai-travel/tech-challenge/internal/chat/model"
	"github.com/openai/openai-go/v2"
)

type Assistant struct {
	cli   openai.Client
	tools *tools.Registry
}

func New() *Assistant {
	return &Assistant{
		cli:   openai.NewClient(),
		tools: tools.Default(),
	}
}

func (a *Assistant) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "An empty conversation", nil
	}

	slog.InfoContext(ctx, "Generating title for conversation", "conversation_id", conv.ID)

	// Task 1: Fix title generation
	// Title generation uses a dedicated system prompt plus the user's opening message.
	// The system message must stay separate from user content: previously the instruction
	// was written to msgs[0] and then overwritten by the user message, so the model
	// answered the question instead of producing a title (e.g. a truncated weather reply).
	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(`Generate a short, descriptive title for a conversation based on the user's opening message. Return only the title text with no quotes or extra punctuation.

Examples:
- "What is the weather like in Barcelona?" → Weather in Barcelona
- "Help me plan a trip to Japan" → Trip to Japan

The title must be a single line, at most 80 characters, with no special characters or emojis.`),
	}

	// Append user messages after the system prompt; only the opening user turn is
	// present when StartConversation calls Title.
	for _, m := range conv.Messages {
		if m.Role == model.RoleUser {
			msgs = append(msgs, openai.UserMessage(m.Content))
		}
	}

	// GPT-4.1 follows short system instructions reliably; o1 was slower and treated
	// the lone user message as a chat question when the system prompt was lost.
	resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModelGPT4_1,
		Messages: msgs,
	})

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 || strings.TrimSpace(resp.Choices[0].Message.Content) == "" {
		return "", errors.New("empty response from OpenAI for title generation")
	}

	title := resp.Choices[0].Message.Content
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.Trim(title, " \t\r\n-\"'")

	if len(title) > 80 {
		title = title[:80]
	}

	return title, nil
}

func (a *Assistant) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "", errors.New("conversation has no messages")
	}

	slog.InfoContext(ctx, "Generating reply for conversation", "conversation_id", conv.ID)

	now := time.Now().Format(time.RFC3339)
	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(fmt.Sprintf(`You are a helpful, concise AI assistant. Today's date and time is %s.

For any weather-related question, you MUST call get_weather before answering. Never rely on memorized or outdated weather information. Base weather answers only on data returned by get_weather.

For forecast questions, call get_weather with days set to the requested number (1-14). For seasonal or long-range questions (e.g. monsoon outlook), still call get_weather with days=14 for the location, then summarize that data and clearly state that only current conditions and up to a 14-day forecast are available—you cannot provide official seasonal monsoon predictions from memory.

For currency conversion or travel budget questions, use get_exchange_rate.`, now)),
	}

	for _, m := range conv.Messages {
		switch m.Role {
		case model.RoleUser:
			msgs = append(msgs, openai.UserMessage(m.Content))
		case model.RoleAssistant:
			msgs = append(msgs, openai.AssistantMessage(m.Content))
		}
	}

	for i := 0; i < 15; i++ {
		resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    openai.ChatModelGPT4_1,
			Messages: msgs,
			Tools:    a.tools.Definitions(),
		})

		if err != nil {
			return "", err
		}

		if len(resp.Choices) == 0 {
			return "", errors.New("no choices returned by OpenAI")
		}

		if message := resp.Choices[0].Message; len(message.ToolCalls) > 0 {
			msgs = append(msgs, message.ToParam())

			for _, call := range message.ToolCalls {
				slog.InfoContext(ctx, "Tool call received", "name", call.Function.Name, "args", call.Function.Arguments)

				result, err := a.tools.Run(ctx, call.Function.Name, call.Function.Arguments)
				if err != nil {
					if errors.Is(err, tools.ErrUnknownTool) {
						return "", err
					}

					msgs = append(msgs, openai.ToolMessage(err.Error(), call.ID))
					continue
				}

				msgs = append(msgs, openai.ToolMessage(result, call.ID))
			}

			continue
		}

		return resp.Choices[0].Message.Content, nil
	}

	return "", errors.New("too many tool calls, unable to generate reply")
}
