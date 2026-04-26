package chats

import (
	"context"

	"code-code.internal/go-contract/agui"
	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"google.golang.org/protobuf/types/known/structpb"
)

func loadAGUIMessages(ctx context.Context, chats chatService, chatID string) ([]aguitypes.Message, error) {
	messages, _, err := chats.ListChatMessages(ctx, chatID, 0, "")
	if err != nil {
		return nil, err
	}
	return buildAGUIMessages(messages)
}

func buildAGUIMessages(messages []*structpb.Struct) ([]aguitypes.Message, error) {
	out := make([]aguitypes.Message, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			continue
		}
		item, err := agui.MessageFromStruct(message)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}
