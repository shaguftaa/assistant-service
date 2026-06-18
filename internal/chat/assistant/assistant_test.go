package assistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/acai-travel/tech-challenge/internal/chat/model"
	"github.com/acai-travel/tech-challenge/internal/chat/assistant/tools"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

func TestAssistant_TitleGeneratesTitleFromOpenAI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		messages, ok := payload["messages"].([]any)
		if !ok || len(messages) != 2 {
			t.Fatalf("expected 2 messages, got %#v", payload["messages"])
		}

		user, ok := messages[1].(map[string]any)
		if !ok {
			t.Fatalf("expected second message to be object, got %#v", messages[1])
		}

		if user["role"] != "user" {
			t.Fatalf("expected second message role user, got %v", user["role"])
		}

		if user["content"] != "What is the weather like in Barcelona?" {
			t.Fatalf("unexpected user content: %v", user["content"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": 1,
			"model":   "gpt-4.1",
			"choices": []any{map[string]any{
				"index":         0,
				"finish_reason": "stop",
				"message": map[string]any{
					"role":    "assistant",
					"content": "Weather in Barcelona",
				},
			}},
		})
	}))
	defer server.Close()

	cli := openai.Client{
		Chat: openai.NewChatService(
			option.WithBaseURL(server.URL),
			option.WithHTTPClient(server.Client()),
		),
	}
	assist := &Assistant{cli: cli, tools: tools.Default()}

	conv := &model.Conversation{
		Messages: []*model.Message{{Role: model.RoleUser, Content: "What is the weather like in Barcelona?"}},
	}

	title, err := assist.Title(context.Background(), conv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if title != "Weather in Barcelona" {
		t.Fatalf("expected title %q, got %q", "Weather in Barcelona", title)
	}
}

func TestAssistant_TitleReturnsDefaultForEmptyConversation(t *testing.T) {
	assist := &Assistant{cli: openai.Client{}, tools: tools.Default()}

	title, err := assist.Title(context.Background(), &model.Conversation{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if title != "An empty conversation" {
		t.Fatalf("expected default empty conversation title, got %q", title)
	}
}
