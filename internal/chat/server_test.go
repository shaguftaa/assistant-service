package chat

import (
	"context"
	"testing"

	"github.com/acai-travel/tech-challenge/internal/chat/model"
	. "github.com/acai-travel/tech-challenge/internal/chat/testing"
	"github.com/acai-travel/tech-challenge/internal/pb"
	"github.com/google/go-cmp/cmp"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/testing/protocmp"
)

type fakeAssistant struct {
	title string
	reply string
}

func (f *fakeAssistant) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	return f.title, nil
}

func (f *fakeAssistant) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	return f.reply, nil
}

func TestServer_StartConversation(t *testing.T) {
	ctx := context.Background()

	t.Run("start conversation creates conversation with title and reply", WithFixture(func(t *testing.T, f *Fixture) {
		assistant := &fakeAssistant{
			title: "Travel to Tokyo",
			reply: "Sure, I can help you plan a trip to Tokyo.",
		}
		srv := NewServer(f.Repository, assistant)

		req := &pb.StartConversationRequest{Message: "I want to travel to Tokyo."}
		resp, err := srv.StartConversation(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.GetConversationId() == "" {
			t.Fatal("expected conversation id")
		}

		if resp.GetTitle() != assistant.title {
			t.Fatalf("expected title %q, got %q", assistant.title, resp.GetTitle())
		}

		if resp.GetReply() != assistant.reply {
			t.Fatalf("expected reply %q, got %q", assistant.reply, resp.GetReply())
		}

		defer func() {
			if err := f.Repository.DeleteConversation(ctx, resp.GetConversationId()); err != nil {
				t.Fatalf("cleanup failed: %v", err)
			}
		}()

		conv, err := f.Repository.DescribeConversation(ctx, resp.GetConversationId())
		if err != nil {
			t.Fatalf("failed to load created conversation: %v", err)
		}

		if conv.Title != assistant.title {
			t.Fatalf("stored conversation title mismatch: expected %q, got %q", assistant.title, conv.Title)
		}

		if len(conv.Messages) != 2 {
			t.Fatalf("expected 2 messages in conversation, got %d", len(conv.Messages))
		}

		if conv.Messages[0].Role != model.RoleUser || conv.Messages[0].Content != req.GetMessage() {
			t.Fatalf("expected first message to be user message %q, got role=%v content=%q", req.GetMessage(), conv.Messages[0].Role, conv.Messages[0].Content)
		}

		if conv.Messages[1].Role != model.RoleAssistant || conv.Messages[1].Content != assistant.reply {
			t.Fatalf("expected second message to be assistant reply %q, got role=%v content=%q", assistant.reply, conv.Messages[1].Role, conv.Messages[1].Content)
		}
	}))
}

func TestServer_DescribeConversation(t *testing.T) {
	ctx := context.Background()
	srv := NewServer(model.New(ConnectMongo()), nil)

	t.Run("describe existing conversation", WithFixture(func(t *testing.T, f *Fixture) {
		c := f.CreateConversation()

		out, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{ConversationId: c.ID.Hex()})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, want := out.GetConversation(), c.Proto()
		if !cmp.Equal(got, want, protocmp.Transform()) {
			t.Errorf("DescribeConversation() mismatch (-got +want):\n%s", cmp.Diff(got, want, protocmp.Transform()))
		}
	}))

	t.Run("describe non existing conversation should return 404", WithFixture(func(t *testing.T, f *Fixture) {
		_, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{ConversationId: "08a59244257c872c5943e2a2"})
		if err == nil {
			t.Fatal("expected error for non-existing conversation, got nil")
		}

		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.NotFound {
			t.Fatalf("expected twirp.NotFound error, got %v", err)
		}
	}))
}
