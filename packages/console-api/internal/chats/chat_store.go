package chats

import (
	"context"

	chatv1 "code-code.internal/go-contract/platform/chat/v1"
	"google.golang.org/protobuf/proto"
)

type chatStore interface {
	Create(context.Context, *chatv1.Chat) (*chatv1.Chat, error)
	Get(context.Context, string) (*chatv1.Chat, error)
	Update(context.Context, *chatv1.Chat) (*chatv1.Chat, error)
	List(context.Context, string, int32, string) ([]*chatv1.Chat, string, error)
}

func cloneChat(chat *chatv1.Chat) *chatv1.Chat {
	if chat == nil {
		return nil
	}
	return proto.Clone(chat).(*chatv1.Chat)
}

func storedChat(chat *chatv1.Chat) *chatv1.Chat {
	next := cloneChat(chat)
	if next == nil {
		return nil
	}
	next.SessionState = nil
	return next
}
