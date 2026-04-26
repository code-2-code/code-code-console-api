package chats

import (
	"context"
	"encoding/json"
	"strings"

	"code-code.internal/go-contract/agui"
	chatv1 "code-code.internal/go-contract/platform/chat/v1"
	sessiondomain "code-code.internal/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func (s *GRPCChatServer) ListChatMessages(ctx context.Context, request *chatv1.ListChatMessagesRequest) (*chatv1.ListChatMessagesResponse, error) {
	if err := s.requireChatState(); err != nil {
		return nil, err
	}
	chatID := strings.TrimSpace(request.GetChatId())
	if chatID == "" {
		return nil, status.Error(codes.InvalidArgument, "chat_id is required")
	}
	chat, err := s.state.chats.Get(ctx, chatID)
	if err != nil {
		return nil, err
	}
	repository, ok := s.state.sessions.(sessiondomain.TurnMessageRepository)
	if !ok {
		return nil, status.Error(codes.Unavailable, "chat transcript repository is not configured")
	}
	messages, nextPageToken, err := repository.ListTurnMessages(ctx, currentSessionID(chatID, chat), request.GetPageSize(), request.GetPageToken())
	if err != nil {
		return nil, err
	}
	response := &chatv1.ListChatMessagesResponse{
		Messages:      make([]*structpb.Struct, 0, len(messages)),
		NextPageToken: nextPageToken,
	}
	for _, message := range messages {
		item, err := rawJSONToProtoStruct(message.Message)
		if err != nil {
			return nil, err
		}
		response.Messages = append(response.Messages, item)
	}
	return response, nil
}

func (c *GRPCChatClient) ListChatMessages(ctx context.Context, chatID string, pageSize int32, pageToken string) ([]*structpb.Struct, string, error) {
	if c == nil || c.client == nil {
		return nil, "", status.Error(codes.Unavailable, "chat client is not configured")
	}
	response, err := c.client.ListChatMessages(ctx, &chatv1.ListChatMessagesRequest{
		ChatId:    chatID,
		PageSize:  pageSize,
		PageToken: pageToken,
	})
	if err != nil {
		return nil, "", err
	}
	return response.GetMessages(), response.GetNextPageToken(), nil
}

func rawJSONToProtoStruct(raw json.RawMessage) (*structpb.Struct, error) {
	payload, err := agui.MessageStructFromRaw(raw)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "ag-ui message is invalid: %v", err)
	}
	return payload, nil
}
