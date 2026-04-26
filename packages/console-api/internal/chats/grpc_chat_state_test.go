package chats

import (
	"context"
	"sort"
	"testing"

	agentsessionv1 "code-code.internal/go-contract/platform/agent_session/v1"
	chatv1 "code-code.internal/go-contract/platform/chat/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCChatServerStoresMetadataAndUsesSessionController(t *testing.T) {
	ctx := context.Background()
	store := newFakeChatStore()
	sessions := newFakeSessions()
	server := NewGRPCChatServer(nil, newDirectState(store, sessions))

	created, err := server.CreateChat(ctx, &chatv1.CreateChatRequest{
		ChatId:      "chat-1",
		ScopeId:     "scope-1",
		DisplayName: "n8n session",
		Session: &agentsessionv1.AgentSessionSpec{
			SessionId: "session-1",
			ProfileId: "profile-1",
		},
	})
	if err != nil {
		t.Fatalf("CreateChat() error = %v", err)
	}
	if got, want := sessions.lastCreateSessionID, "session-1"; got != want {
		t.Fatalf("created session_id = %q, want %q", got, want)
	}
	if got, want := sessions.createCount, 1; got != want {
		t.Fatalf("create session count = %d, want %d", got, want)
	}
	if got, want := created.GetChat().GetChatId(), "chat-1"; got != want {
		t.Fatalf("chat_id = %q, want %q", got, want)
	}
	if got, want := created.GetChat().GetSessionId(), "session-1"; got != want {
		t.Fatalf("chat session_id = %q, want %q", got, want)
	}
	if got, want := store.items["chat-1"].GetDisplayName(), "n8n session"; got != want {
		t.Fatalf("stored display_name = %q, want %q", got, want)
	}
	if store.items["chat-1"].GetSessionState() != nil {
		t.Fatalf("stored chat must not persist session state")
	}

	got, err := server.GetChat(ctx, &chatv1.GetChatRequest{ChatId: "chat-1"})
	if err != nil {
		t.Fatalf("GetChat() error = %v", err)
	}
	if got.GetChat().GetSessionState() == nil {
		t.Fatalf("GetChat() should include live session state")
	}
	if got, want := sessions.lastGetSessionID, "session-1"; got != want {
		t.Fatalf("GetChat() session_id = %q, want %q", got, want)
	}
	if got.GetChat().GetDisplayName() != "n8n session" {
		t.Fatalf("display_name = %q, want n8n session", got.GetChat().GetDisplayName())
	}

	updated, err := server.UpdateChatSessionSetup(ctx, &chatv1.UpdateChatSessionSetupRequest{
		ChatId: "chat-1",
		Session: &agentsessionv1.AgentSessionSpec{
			SessionId: "session-1",
			ProfileId: "profile-1",
		},
	})
	if err != nil {
		t.Fatalf("UpdateChatSessionSetup() error = %v", err)
	}
	if got, want := sessions.lastUpdateSessionID, "session-1"; got != want {
		t.Fatalf("updated session_id = %q, want %q", got, want)
	}
	if updated.GetChat().GetSessionState() == nil {
		t.Fatalf("UpdateChatSessionSetup() should include live session state")
	}
	if got, want := updated.GetChat().GetDisplayName(), "n8n session"; got != want {
		t.Fatalf("setup update display_name = %q, want %q", got, want)
	}

	renamed, err := server.RenameChat(ctx, &chatv1.RenameChatRequest{
		ChatId:      "chat-1",
		DisplayName: "renamed session",
	})
	if err != nil {
		t.Fatalf("RenameChat() error = %v", err)
	}
	if got, want := renamed.GetChat().GetDisplayName(), "renamed session"; got != want {
		t.Fatalf("renamed display_name = %q, want %q", got, want)
	}
	if got, want := store.items["chat-1"].GetDisplayName(), "renamed session"; got != want {
		t.Fatalf("stored renamed display_name = %q, want %q", got, want)
	}
	if got, want := sessions.lastUpdateSessionID, "session-1"; got != want {
		t.Fatalf("rename should not update session, last update session_id = %q, want %q", got, want)
	}

	_, err = server.CreateChat(ctx, &chatv1.CreateChatRequest{
		ChatId: "chat-1",
		Session: &agentsessionv1.AgentSessionSpec{
			SessionId: "session-2",
			ProfileId: "profile-1",
		},
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate CreateChat() code = %v, want AlreadyExists", status.Code(err))
	}
	if got, want := sessions.createCount, 1; got != want {
		t.Fatalf("duplicate create session count = %d, want %d", got, want)
	}

	list, err := server.ListChats(ctx, &chatv1.ListChatsRequest{ScopeId: "scope-1"})
	if err != nil {
		t.Fatalf("ListChats() error = %v", err)
	}
	if len(list.GetChats()) != 1 {
		t.Fatalf("ListChats() length = %d, want 1", len(list.GetChats()))
	}
	if list.GetChats()[0].GetSessionState() != nil {
		t.Fatalf("ListChats() should not include live session state")
	}
}

type fakeChatStore struct {
	items map[string]*chatv1.Chat
}

func newFakeChatStore() *fakeChatStore {
	return &fakeChatStore{items: map[string]*chatv1.Chat{}}
}

func (s *fakeChatStore) Create(_ context.Context, chat *chatv1.Chat) (*chatv1.Chat, error) {
	if _, ok := s.items[chat.GetChatId()]; ok {
		return nil, status.Error(codes.AlreadyExists, "chat already exists")
	}
	next := storedChat(chat)
	s.items[next.GetChatId()] = next
	return cloneChat(next), nil
}

func (s *fakeChatStore) Get(_ context.Context, chatID string) (*chatv1.Chat, error) {
	chat, ok := s.items[chatID]
	if !ok {
		return nil, status.Error(codes.NotFound, "chat not found")
	}
	return cloneChat(chat), nil
}

func (s *fakeChatStore) Update(_ context.Context, chat *chatv1.Chat) (*chatv1.Chat, error) {
	if _, ok := s.items[chat.GetChatId()]; !ok {
		return nil, status.Error(codes.NotFound, "chat not found")
	}
	next := storedChat(chat)
	s.items[next.GetChatId()] = next
	return cloneChat(next), nil
}

func (s *fakeChatStore) List(_ context.Context, scopeID string, _ int32, _ string) ([]*chatv1.Chat, string, error) {
	ids := make([]string, 0, len(s.items))
	for id, chat := range s.items {
		if scopeID == "" || chat.GetScopeId() == scopeID {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	items := make([]*chatv1.Chat, 0, len(ids))
	for _, id := range ids {
		items = append(items, cloneChat(s.items[id]))
	}
	return items, "", nil
}
