package chats

import (
	"context"
	"strings"

	agentsessionv1 "code-code.internal/go-contract/platform/agent_session/v1"
	chatv1 "code-code.internal/go-contract/platform/chat/v1"
	sessiondomain "code-code.internal/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultChatScopeID = "default"

func (s *GRPCChatServer) CreateChat(ctx context.Context, request *chatv1.CreateChatRequest) (*chatv1.CreateChatResponse, error) {
	if err := s.requireChatState(); err != nil {
		return nil, err
	}
	chatID := strings.TrimSpace(request.GetChatId())
	if chatID == "" {
		return nil, status.Error(codes.InvalidArgument, "chat_id is required")
	}
	var response *chatv1.CreateChatResponse
	if err := s.state.setup.Do(ctx, func(chats chatStore, sessions sessiondomain.Repository) error {
		if _, err := chats.Get(ctx, chatID); err == nil {
			return status.Error(codes.AlreadyExists, "chat already exists")
		} else if status.Code(err) != codes.NotFound {
			return err
		}
		sessionID := firstNonEmpty(request.GetSession().GetSessionId(), chatID)
		sessionSpec, err := sessiondomain.NormalizeSpec(sessionID, request.GetSession())
		if err != nil {
			return err
		}
		session, err := sessions.Create(ctx, sessionSpec)
		if err != nil {
			return err
		}
		now := timestamppb.Now()
		stored, err := chats.Create(ctx, &chatv1.Chat{
			ChatId:      chatID,
			ScopeId:     firstNonEmpty(strings.TrimSpace(request.GetScopeId()), defaultChatScopeID),
			DisplayName: strings.TrimSpace(request.GetDisplayName()),
			SessionId:   sessionSpec.GetSessionId(),
			CreatedAt:   now,
			UpdatedAt:   now,
		})
		if err != nil {
			return err
		}
		response = &chatv1.CreateChatResponse{Chat: chatWithSession(stored, session)}
		return nil
	}); err != nil {
		return nil, err
	}
	return response, nil
}

func (s *GRPCChatServer) GetChat(ctx context.Context, request *chatv1.GetChatRequest) (*chatv1.GetChatResponse, error) {
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
	sessionID := currentSessionID(chatID, chat)
	session, err := s.state.sessions.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return &chatv1.GetChatResponse{Chat: chatWithSession(chat, session)}, nil
}

func (s *GRPCChatServer) UpdateChatSessionSetup(ctx context.Context, request *chatv1.UpdateChatSessionSetupRequest) (*chatv1.UpdateChatSessionSetupResponse, error) {
	if err := s.requireChatState(); err != nil {
		return nil, err
	}
	chatID := strings.TrimSpace(request.GetChatId())
	if chatID == "" {
		return nil, status.Error(codes.InvalidArgument, "chat_id is required")
	}
	var response *chatv1.UpdateChatSessionSetupResponse
	if err := s.state.setup.Do(ctx, func(chats chatStore, sessions sessiondomain.Repository) error {
		current, err := chats.Get(ctx, chatID)
		if err != nil {
			return err
		}
		sessionID := currentSessionID(chatID, current)
		currentSession, err := sessions.Get(ctx, sessionID)
		if err != nil {
			return err
		}
		sessionSpec, err := sessiondomain.NormalizeSpec(sessionID, request.GetSession())
		if err != nil {
			return err
		}
		updatedSession := currentSession
		if !proto.Equal(currentSession.GetSpec(), sessionSpec) {
			updatedSession, err = sessions.Update(ctx, sessionID, sessionSpec)
			if err != nil {
				return err
			}
		}
		response = &chatv1.UpdateChatSessionSetupResponse{Chat: chatWithSession(current, updatedSession)}
		return nil
	}); err != nil {
		return nil, err
	}
	return response, nil
}

func (s *GRPCChatServer) RenameChat(ctx context.Context, request *chatv1.RenameChatRequest) (*chatv1.RenameChatResponse, error) {
	if err := s.requireChatState(); err != nil {
		return nil, err
	}
	chatID := strings.TrimSpace(request.GetChatId())
	if chatID == "" {
		return nil, status.Error(codes.InvalidArgument, "chat_id is required")
	}
	displayName := strings.TrimSpace(request.GetDisplayName())
	if displayName == "" {
		return nil, status.Error(codes.InvalidArgument, "display_name is required")
	}
	current, err := s.state.chats.Get(ctx, chatID)
	if err != nil {
		return nil, err
	}
	current.DisplayName = displayName
	stored, err := s.state.chats.Update(ctx, current)
	if err != nil {
		return nil, err
	}
	session, err := s.state.sessions.Get(ctx, currentSessionID(chatID, stored))
	if err != nil {
		return nil, err
	}
	return &chatv1.RenameChatResponse{Chat: chatWithSession(stored, session)}, nil
}

func (s *GRPCChatServer) ListChats(ctx context.Context, request *chatv1.ListChatsRequest) (*chatv1.ListChatsResponse, error) {
	if err := s.requireChatState(); err != nil {
		return nil, err
	}
	items, nextPageToken, err := s.state.chats.List(ctx, request.GetScopeId(), request.GetPageSize(), request.GetPageToken())
	if err != nil {
		return nil, err
	}
	return &chatv1.ListChatsResponse{
		Chats:         cloneChats(items),
		NextPageToken: nextPageToken,
	}, nil
}

func (s *GRPCChatServer) requireChatState() error {
	if s == nil || s.state == nil || s.state.chats == nil || s.state.sessions == nil || s.state.setup == nil {
		return status.Error(codes.Unavailable, "chat state service is not configured")
	}
	return nil
}

func cloneChats(items []*chatv1.Chat) []*chatv1.Chat {
	if len(items) == 0 {
		return nil
	}
	out := make([]*chatv1.Chat, 0, len(items))
	for _, item := range items {
		out = append(out, cloneChat(item))
	}
	return out
}

func chatWithSession(chat *chatv1.Chat, session *agentsessionv1.AgentSessionState) *chatv1.Chat {
	next := cloneChat(chat)
	if next == nil {
		return nil
	}
	next.SessionState = cloneSessionState(session)
	return next
}

func cloneSessionState(session *agentsessionv1.AgentSessionState) *agentsessionv1.AgentSessionState {
	if session == nil {
		return nil
	}
	return proto.Clone(session).(*agentsessionv1.AgentSessionState)
}
