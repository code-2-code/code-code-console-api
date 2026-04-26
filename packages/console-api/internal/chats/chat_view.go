package chats

import (
	"encoding/json"
	"fmt"
	"strings"

	capv1 "code-code.internal/go-contract/agent/cap/v1"
	agentsessionv1 "code-code.internal/go-contract/platform/agent_session/v1"
	chatv1 "code-code.internal/go-contract/platform/chat/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	chatModeProfile = "profile"
	chatModeInline  = "inline"
)

type chatView struct {
	ID          string          `json:"id"`
	DisplayName string          `json:"displayName,omitempty"`
	Session     chatSessionView `json:"session"`
}

type chatSessionView struct {
	ID           string                `json:"id"`
	SessionSetup chatSetupView         `json:"sessionSetup"`
	State        aguiProjectionSession `json:"state"`
}

type chatListView struct {
	Items         []chatSummaryView `json:"items"`
	NextPageToken string            `json:"nextPageToken,omitempty"`
}

type chatSummaryView struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName,omitempty"`
	SessionID   string `json:"sessionId,omitempty"`
}

type chatSetupView struct {
	Mode           string          `json:"mode"`
	ProfileID      string          `json:"profileId,omitempty"`
	ProviderID     string          `json:"providerId,omitempty"`
	ExecutionClass string          `json:"executionClass,omitempty"`
	Editable       bool            `json:"editable"`
	RuntimeConfig  json.RawMessage `json:"runtimeConfig,omitempty"`
	ResourceConfig json.RawMessage `json:"resourceConfig,omitempty"`
}

func buildChatView(chat *chatv1.Chat) (*chatView, error) {
	if chat == nil {
		return nil, fmt.Errorf("chat is nil")
	}
	chatID := firstNonEmpty(chat.GetChatId(), chat.GetSessionId())
	spec := currentSessionSpec(chat)
	mode := sessionMode(spec)
	setup, err := buildChatSetupView(spec, mode)
	if err != nil {
		return nil, err
	}
	sessionID := currentSessionID(chatID, chat)
	projection := buildAGUIProjection(sessionID, chat.GetSessionState(), nil)
	return &chatView{
		ID:          chatID,
		DisplayName: strings.TrimSpace(chat.GetDisplayName()),
		Session: chatSessionView{
			ID:           sessionID,
			SessionSetup: setup,
			State:        projection.Session,
		},
	}, nil
}

func buildChatListView(chats []*chatv1.Chat, nextPageToken string) (*chatListView, error) {
	view := &chatListView{
		Items:         make([]chatSummaryView, 0, len(chats)),
		NextPageToken: strings.TrimSpace(nextPageToken),
	}
	for _, chat := range chats {
		view.Items = append(view.Items, chatSummaryView{
			ID:          strings.TrimSpace(chat.GetChatId()),
			DisplayName: strings.TrimSpace(chat.GetDisplayName()),
			SessionID:   currentSessionID(chat.GetChatId(), chat),
		})
	}
	return view, nil
}

func sessionMode(spec *agentsessionv1.AgentSessionSpec) string {
	if spec != nil && strings.TrimSpace(spec.GetProfileId()) != "" {
		return chatModeProfile
	}
	return chatModeInline
}

func buildChatSetupView(spec *agentsessionv1.AgentSessionSpec, mode string) (chatSetupView, error) {
	if spec == nil {
		return chatSetupView{}, fmt.Errorf("chat session spec is nil")
	}
	view := chatSetupView{
		Mode:           mode,
		ProfileID:      strings.TrimSpace(spec.GetProfileId()),
		ProviderID:     strings.TrimSpace(spec.GetProviderId()),
		ExecutionClass: strings.TrimSpace(spec.GetExecutionClass()),
		Editable:       mode == chatModeInline,
	}
	if mode != chatModeInline {
		return view, nil
	}
	runtimeConfig, err := marshalProtoJSON(spec.GetRuntimeConfig())
	if err != nil {
		return chatSetupView{}, err
	}
	resourceConfig, err := marshalProtoJSON(cloneResourceConfig(spec.GetResourceConfig()))
	if err != nil {
		return chatSetupView{}, err
	}
	view.RuntimeConfig = runtimeConfig
	view.ResourceConfig = resourceConfig
	return view, nil
}

func marshalProtoJSON(message proto.Message) (json.RawMessage, error) {
	if message == nil {
		return nil, nil
	}
	data, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(message)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func cloneResourceConfig(config *capv1.AgentResources) *capv1.AgentResources {
	if config == nil {
		return nil
	}
	return proto.Clone(config).(*capv1.AgentResources)
}
