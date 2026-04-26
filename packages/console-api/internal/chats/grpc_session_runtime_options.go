package chats

import (
	"context"
	"fmt"

	agentsessionv1 "code-code.internal/go-contract/platform/agent_session/v1"
	chatv1 "code-code.internal/go-contract/platform/chat/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

type GRPCChatServer struct {
	chatv1.UnimplementedChatServiceServer

	sessionRuntimeOptions sessionRuntimeOptionsService
	state                 *State
}

func NewGRPCChatServer(
	sessionRuntimeOptions sessionRuntimeOptionsService,
	state *State,
) *GRPCChatServer {
	return &GRPCChatServer{
		sessionRuntimeOptions: sessionRuntimeOptions,
		state:                 state,
	}
}

func (s *GRPCChatServer) GetSessionRuntimeOptions(ctx context.Context, _ *chatv1.GetSessionRuntimeOptionsRequest) (*chatv1.GetSessionRuntimeOptionsResponse, error) {
	if s == nil || s.sessionRuntimeOptions == nil {
		return nil, fmt.Errorf("consoleapi/chats: session runtime options service is not configured")
	}
	view, err := s.sessionRuntimeOptions.View(ctx)
	if err != nil {
		return nil, err
	}
	return sessionRuntimeOptionsToProto(view), nil
}

func (s *GRPCChatServer) ValidateInlineSpec(ctx context.Context, request *chatv1.ValidateInlineSpecRequest) (*chatv1.ValidateInlineSpecResponse, error) {
	if s == nil || s.sessionRuntimeOptions == nil {
		return nil, fmt.Errorf("consoleapi/chats: session runtime options service is not configured")
	}
	if err := s.sessionRuntimeOptions.ValidateInlineSpec(ctx, request.GetSpec()); err != nil {
		return nil, err
	}
	return &chatv1.ValidateInlineSpecResponse{}, nil
}

type GRPCChatClient struct {
	client chatv1.ChatServiceClient
}

func NewGRPCChatClient(client chatv1.ChatServiceClient) *GRPCChatClient {
	if client == nil {
		return nil
	}
	return &GRPCChatClient{client: client}
}

func (c *GRPCChatClient) View(ctx context.Context) (*sessionRuntimeOptionsView, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("consoleapi/chats: session runtime options client is not configured")
	}
	response, err := c.client.GetSessionRuntimeOptions(ctx, &chatv1.GetSessionRuntimeOptionsRequest{})
	if err != nil {
		return nil, err
	}
	return sessionRuntimeOptionsFromProto(response), nil
}

func (c *GRPCChatClient) ValidateInlineSpec(ctx context.Context, spec *agentsessionv1.AgentSessionSpec) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("consoleapi/chats: session runtime options client is not configured")
	}
	_, err := c.client.ValidateInlineSpec(ctx, &chatv1.ValidateInlineSpecRequest{Spec: spec})
	return err
}

func (c *GRPCChatClient) GetChat(ctx context.Context, chatID string) (*chatv1.Chat, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("consoleapi/chats: chat client is not configured")
	}
	response, err := c.client.GetChat(ctx, &chatv1.GetChatRequest{ChatId: chatID})
	if err != nil {
		return nil, err
	}
	return response.GetChat(), nil
}

func (c *GRPCChatClient) CreateChat(ctx context.Context, chatID string, scopeID string, displayName string, session *agentsessionv1.AgentSessionSpec) (*chatv1.Chat, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("consoleapi/chats: chat client is not configured")
	}
	response, err := c.client.CreateChat(ctx, &chatv1.CreateChatRequest{
		ChatId:      chatID,
		ScopeId:     scopeID,
		DisplayName: displayName,
		Session:     session,
	})
	if err != nil {
		return nil, err
	}
	return response.GetChat(), nil
}

func (c *GRPCChatClient) UpdateChatSessionSetup(ctx context.Context, chatID string, session *agentsessionv1.AgentSessionSpec) (*chatv1.Chat, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("consoleapi/chats: chat client is not configured")
	}
	response, err := c.client.UpdateChatSessionSetup(ctx, &chatv1.UpdateChatSessionSetupRequest{
		ChatId:  chatID,
		Session: session,
	})
	if err != nil {
		return nil, err
	}
	return response.GetChat(), nil
}

func (c *GRPCChatClient) RenameChat(ctx context.Context, chatID string, displayName string) (*chatv1.Chat, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("consoleapi/chats: chat client is not configured")
	}
	response, err := c.client.RenameChat(ctx, &chatv1.RenameChatRequest{
		ChatId:      chatID,
		DisplayName: displayName,
	})
	if err != nil {
		return nil, err
	}
	return response.GetChat(), nil
}

func (c *GRPCChatClient) ListChats(ctx context.Context, scopeID string, pageSize int32, pageToken string) ([]*chatv1.Chat, string, error) {
	if c == nil || c.client == nil {
		return nil, "", fmt.Errorf("consoleapi/chats: chat client is not configured")
	}
	response, err := c.client.ListChats(ctx, &chatv1.ListChatsRequest{
		ScopeId:   scopeID,
		PageSize:  pageSize,
		PageToken: pageToken,
	})
	if err != nil {
		return nil, "", err
	}
	return response.GetChats(), response.GetNextPageToken(), nil
}

func sessionRuntimeOptionsToProto(view *sessionRuntimeOptionsView) *chatv1.GetSessionRuntimeOptionsResponse {
	response := &chatv1.GetSessionRuntimeOptionsResponse{}
	if view == nil {
		return response
	}
	response.Items = make([]*chatv1.SessionRuntimeProviderOption, 0, len(view.Items))
	for _, item := range view.Items {
		next := &chatv1.SessionRuntimeProviderOption{
			ProviderId:       item.ProviderID,
			Label:            item.Label,
			ExecutionClasses: append([]string(nil), item.ExecutionClasses...),
			Surfaces:         make([]*chatv1.SessionRuntimeSurfaceOption, 0, len(item.Surfaces)),
		}
		for _, surface := range item.Surfaces {
			runtimeRef := cloneRuntimeRef(surface.RuntimeRef)
			next.Surfaces = append(next.Surfaces, &chatv1.SessionRuntimeSurfaceOption{
				RuntimeRef: runtimeRef,
				Label:      surface.Label,
				Models:     append([]string(nil), surface.Models...),
			})
		}
		response.Items = append(response.Items, next)
	}
	return response
}

func sessionRuntimeOptionsFromProto(response *chatv1.GetSessionRuntimeOptionsResponse) *sessionRuntimeOptionsView {
	view := &sessionRuntimeOptionsView{}
	if response == nil {
		return view
	}
	view.Items = make([]sessionRuntimeProviderOption, 0, len(response.GetItems()))
	for _, item := range response.GetItems() {
		next := sessionRuntimeProviderOption{
			ProviderID:       item.GetProviderId(),
			Label:            item.GetLabel(),
			ExecutionClasses: append([]string(nil), item.GetExecutionClasses()...),
			Surfaces:         make([]sessionRuntimeSurfaceOption, 0, len(item.GetSurfaces())),
		}
		for _, surface := range item.GetSurfaces() {
			next.Surfaces = append(next.Surfaces, sessionRuntimeSurfaceOption{
				RuntimeRef: cloneRuntimeRef(surface.GetRuntimeRef()),
				Label:      surface.GetLabel(),
				Models:     append([]string(nil), surface.GetModels()...),
			})
		}
		view.Items = append(view.Items, next)
	}
	return view
}

func cloneRuntimeRef(ref *providerv1.ProviderRuntimeRef) *providerv1.ProviderRuntimeRef {
	if ref == nil {
		return nil
	}
	return proto.Clone(ref).(*providerv1.ProviderRuntimeRef)
}
