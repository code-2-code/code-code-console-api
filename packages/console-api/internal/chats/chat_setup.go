package chats

import (
	"context"
	"encoding/json"
	"strings"

	capv1 "code-code.internal/go-contract/agent/cap/v1"
	agentsessionv1 "code-code.internal/go-contract/platform/agent_session/v1"
	chatv1 "code-code.internal/go-contract/platform/chat/v1"
	sessiondomain "code-code.internal/session"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

type putChatRequest struct {
	DisplayName  string              `json:"displayName,omitempty"`
	SessionSetup putChatSessionSetup `json:"sessionSetup"`
}

type putChatSessionSetup struct {
	Mode      string                `json:"mode"`
	ProfileID string                `json:"profileId,omitempty"`
	Inline    *putInlineChatRequest `json:"inline,omitempty"`
}

type putInlineChatRequest struct {
	ProviderID     string          `json:"providerId,omitempty"`
	ExecutionClass string          `json:"executionClass,omitempty"`
	RuntimeConfig  json.RawMessage `json:"runtimeConfig,omitempty"`
	ResourceConfig json.RawMessage `json:"resourceConfig,omitempty"`
}

func upsertChat(
	ctx context.Context,
	chats chatService,
	sessionRuntimeOptions sessionRuntimeOptionsService,
	chatID string,
	request putChatRequest,
) (*chatView, error) {
	current, err := chats.GetChat(ctx, chatID)
	if err != nil && grpcstatus.Code(err) != codes.NotFound {
		return nil, err
	}
	exists := err == nil && current != nil
	switch strings.TrimSpace(request.SessionSetup.Mode) {
	case chatModeProfile:
		return upsertProfileChat(ctx, chats, chatID, current, exists, request)
	case chatModeInline:
		return upsertInlineChat(ctx, chats, sessionRuntimeOptions, chatID, current, exists, request)
	default:
		return nil, grpcstatus.Error(codes.InvalidArgument, "mode must be profile or inline")
	}
}

func upsertProfileChat(
	ctx context.Context,
	chats chatService,
	chatID string,
	current *chatv1.Chat,
	exists bool,
	request putChatRequest,
) (*chatView, error) {
	profileID := strings.TrimSpace(request.SessionSetup.ProfileID)
	if profileID == "" {
		return nil, grpcstatus.Error(codes.InvalidArgument, "profileId is required for profile mode")
	}
	if request.SessionSetup.Inline != nil {
		return nil, grpcstatus.Error(codes.InvalidArgument, "inline setup is not allowed for profile mode")
	}
	if exists {
		spec := currentSessionSpec(current)
		if sessionMode(spec) != chatModeProfile {
			return nil, grpcstatus.Error(codes.FailedPrecondition, "chat mode is immutable")
		}
		if strings.TrimSpace(spec.GetProfileId()) != profileID {
			return nil, grpcstatus.Error(codes.FailedPrecondition, "profile-backed chat cannot change profileId")
		}
		return buildChatView(current)
	}
	spec, err := sessiondomain.NewProfileSpec(chatID, profileID)
	if err != nil {
		return nil, err
	}
	created, err := chats.CreateChat(ctx, chatID, "", request.DisplayName, spec)
	if err != nil {
		return nil, err
	}
	return buildChatView(created)
}

func upsertInlineChat(
	ctx context.Context,
	chats chatService,
	sessionRuntimeOptions sessionRuntimeOptionsService,
	chatID string,
	current *chatv1.Chat,
	exists bool,
	request putChatRequest,
) (*chatView, error) {
	inline := request.SessionSetup.Inline
	if inline == nil {
		return nil, grpcstatus.Error(codes.InvalidArgument, "inline setup is required for inline mode")
	}
	if strings.TrimSpace(request.SessionSetup.ProfileID) != "" {
		return nil, grpcstatus.Error(codes.InvalidArgument, "profileId is not allowed for inline mode")
	}
	if exists {
		spec := currentSessionSpec(current)
		if sessionMode(spec) != chatModeInline {
			return nil, grpcstatus.Error(codes.FailedPrecondition, "chat mode is immutable")
		}
		sessionID := currentSessionID(chatID, current)
		next, err := mergedInlineSessionSpec(sessionID, spec, inline)
		if err != nil {
			return nil, err
		}
		if sessionRuntimeOptions != nil {
			if err := sessionRuntimeOptions.ValidateInlineSpec(ctx, next); err != nil {
				return nil, err
			}
		}
		updated, err := chats.UpdateChatSessionSetup(ctx, chatID, next)
		if err != nil {
			return nil, err
		}
		return buildChatView(updated)
	}
	spec, err := inlineSessionSpec(chatID, inline)
	if err != nil {
		return nil, err
	}
	if sessionRuntimeOptions != nil {
		if err := sessionRuntimeOptions.ValidateInlineSpec(ctx, spec); err != nil {
			return nil, err
		}
	}
	created, err := chats.CreateChat(ctx, chatID, "", request.DisplayName, spec)
	if err != nil {
		return nil, err
	}
	return buildChatView(created)
}

func currentSessionSpec(chat *chatv1.Chat) *agentsessionv1.AgentSessionSpec {
	if chat == nil {
		return nil
	}
	if chat.GetSessionState() != nil {
		return chat.GetSessionState().GetSpec()
	}
	return nil
}

func currentSessionID(chatID string, chat *chatv1.Chat) string {
	if chat == nil {
		return strings.TrimSpace(chatID)
	}
	return firstNonEmpty(chat.GetSessionId(), chat.GetSessionState().GetSpec().GetSessionId(), chatID)
}

func inlineSessionSpec(chatID string, request *putInlineChatRequest) (*agentsessionv1.AgentSessionSpec, error) {
	if request == nil {
		return nil, grpcstatus.Error(codes.InvalidArgument, "inline setup is required")
	}
	providerID := strings.TrimSpace(request.ProviderID)
	if providerID == "" {
		return nil, grpcstatus.Error(codes.InvalidArgument, "inline.providerId is required")
	}
	executionClass := strings.TrimSpace(request.ExecutionClass)
	if executionClass == "" {
		return nil, grpcstatus.Error(codes.InvalidArgument, "inline.executionClass is required")
	}
	runtimeConfig, err := decodeRuntimeConfig(request.RuntimeConfig)
	if err != nil {
		return nil, err
	}
	resourceConfig, err := decodeResourceConfig(request.ResourceConfig)
	if err != nil {
		return nil, err
	}
	return sessiondomain.NormalizeSpec(chatID, &agentsessionv1.AgentSessionSpec{
		ProviderId:     providerID,
		ExecutionClass: executionClass,
		RuntimeConfig:  runtimeConfig,
		ResourceConfig: resourceConfig,
	})
}

func mergedInlineSessionSpec(sessionID string, current *agentsessionv1.AgentSessionSpec, request *putInlineChatRequest) (*agentsessionv1.AgentSessionSpec, error) {
	if current == nil {
		return nil, grpcstatus.Error(codes.InvalidArgument, "current inline session is missing")
	}
	providerID := strings.TrimSpace(current.GetProviderId())
	if nextProviderID := strings.TrimSpace(request.ProviderID); nextProviderID != "" && nextProviderID != providerID {
		return nil, grpcstatus.Error(codes.FailedPrecondition, "inline chat cannot change providerId")
	}
	executionClass := strings.TrimSpace(current.GetExecutionClass())
	if nextExecutionClass := strings.TrimSpace(request.ExecutionClass); nextExecutionClass != "" && nextExecutionClass != executionClass {
		return nil, grpcstatus.Error(codes.FailedPrecondition, "inline chat cannot change executionClass")
	}
	runtimeConfig := current.GetRuntimeConfig()
	if hasJSONBody(request.RuntimeConfig) {
		nextRuntimeConfig, err := decodeRuntimeConfig(request.RuntimeConfig)
		if err != nil {
			return nil, err
		}
		runtimeConfig = nextRuntimeConfig
	}
	resourceConfig := current.GetResourceConfig()
	if hasJSONBody(request.ResourceConfig) {
		nextResourceConfig, err := decodeResourceConfig(request.ResourceConfig)
		if err != nil {
			return nil, err
		}
		resourceConfig = nextResourceConfig
	}
	return sessiondomain.NormalizeSpec(sessionID, &agentsessionv1.AgentSessionSpec{
		ProviderId:     providerID,
		ExecutionClass: executionClass,
		RuntimeConfig:  sessiondomain.CloneRuntimeConfig(runtimeConfig),
		ResourceConfig: sessiondomain.CloneAndNormalizeResourceConfig(resourceConfig),
		WorkspaceRef:   sessiondomain.CloneWorkspaceRef(current.GetWorkspaceRef(), sessiondomain.DefaultWorkspaceRef(sessionID)),
		HomeStateRef:   sessiondomain.CloneHomeStateRef(current.GetHomeStateRef(), sessiondomain.DefaultHomeStateRef(sessionID)),
		PrepareJobs:    sessiondomain.ClonePrepareJobs(current.GetPrepareJobs()),
	})
}

func decodeRuntimeConfig(raw json.RawMessage) (*agentsessionv1.AgentSessionRuntimeConfig, error) {
	if !hasJSONBody(raw) {
		return nil, grpcstatus.Error(codes.InvalidArgument, "inline.runtimeConfig is required")
	}
	config := &agentsessionv1.AgentSessionRuntimeConfig{}
	unmarshal := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := unmarshal.Unmarshal(raw, config); err != nil {
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "inline.runtimeConfig is invalid: %v", err)
	}
	return config, nil
}

func decodeResourceConfig(raw json.RawMessage) (*capv1.AgentResources, error) {
	if !hasJSONBody(raw) {
		return nil, grpcstatus.Error(codes.InvalidArgument, "inline.resourceConfig is required")
	}
	config := &capv1.AgentResources{}
	unmarshal := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := unmarshal.Unmarshal(raw, config); err != nil {
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "inline.resourceConfig is invalid: %v", err)
	}
	return sessiondomain.CloneAndNormalizeResourceConfig(config), nil
}

func hasJSONBody(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed != "" && trimmed != "null"
}
