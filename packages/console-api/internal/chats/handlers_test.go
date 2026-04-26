package chats

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	capv1 "code-code.internal/go-contract/agent/cap/v1"
	agentcorev1 "code-code.internal/go-contract/agent/core/v1"
	resultv1 "code-code.internal/go-contract/agent/result/v1"
	agentrunv1 "code-code.internal/go-contract/platform/agent_run/v1"
	agentsessionv1 "code-code.internal/go-contract/platform/agent_session/v1"
	agentsessionactionv1 "code-code.internal/go-contract/platform/agent_session_action/v1"
	chatv1 "code-code.internal/go-contract/platform/chat/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	runeventv1 "code-code.internal/go-contract/platform/run_event/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type deadlineRecorder struct {
	*httptest.ResponseRecorder
	writeDeadline time.Time
}

func (r *deadlineRecorder) SetWriteDeadline(deadline time.Time) error {
	r.writeDeadline = deadline
	return nil
}

func TestNewAGUIStreamWriterClearsWriteDeadline(t *testing.T) {
	recorder := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}

	stream, err := newAGUIStreamWriter(recorder)
	if err != nil {
		t.Fatalf("newAGUIStreamWriter() error = %v", err)
	}
	if stream == nil {
		t.Fatalf("newAGUIStreamWriter() returned nil stream")
	}
	if !recorder.writeDeadline.IsZero() {
		t.Fatalf("write deadline = %v, want zero", recorder.writeDeadline)
	}
}

func TestRegisterHandlersProfileChatMainline(t *testing.T) {
	mux := http.NewServeMux()
	sessions := newFakeSessions()
	actions := &fakeActions{}
	runs := &fakeRuns{}
	RegisterHandlers(mux, sessions, sessions, actions, runs, nil, nil)

	put := httptest.NewRequest(http.MethodPut, "/api/chats/chat-1", strings.NewReader(`{"sessionSetup":{"mode":"profile","profileId":"profile-1"}}`))
	put.Header.Set("Content-Type", "application/json")
	putRecorder := httptest.NewRecorder()
	mux.ServeHTTP(putRecorder, put)
	if putRecorder.Code != http.StatusOK {
		t.Fatalf("put status = %d, want 200, body=%s", putRecorder.Code, putRecorder.Body.String())
	}
	if got, want := sessions.lastCreateSessionID, "chat-1"; got != want {
		t.Fatalf("create session_id = %q, want %q", got, want)
	}
	if got, want := sessions.lastCreateProfileID, "profile-1"; got != want {
		t.Fatalf("create profile_id = %q, want %q", got, want)
	}
	if got, want := sessions.lastCreateWorkspaceID, "chat-1-workspace"; got != want {
		t.Fatalf("create workspace_id = %q, want %q", got, want)
	}
	if got, want := sessions.lastCreateHomeStateID, "chat-1-home"; got != want {
		t.Fatalf("create home_state_id = %q, want %q", got, want)
	}

	get := httptest.NewRequest(http.MethodGet, "/api/chats/chat-1", nil)
	getRecorder := httptest.NewRecorder()
	mux.ServeHTTP(getRecorder, get)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200, body=%s", getRecorder.Code, getRecorder.Body.String())
	}
	var payload struct {
		ID      string `json:"id"`
		Session struct {
			SessionSetup struct {
				Mode      string `json:"mode"`
				ProfileID string `json:"profileId"`
				Editable  bool   `json:"editable"`
			} `json:"sessionSetup"`
			State struct {
				Phase string `json:"phase"`
			} `json:"state"`
		} `json:"session"`
	}
	if err := json.Unmarshal(getRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got, want := payload.ID, "chat-1"; got != want {
		t.Fatalf("chat id = %q, want %q", got, want)
	}
	if got, want := payload.Session.SessionSetup.Mode, chatModeProfile; got != want {
		t.Fatalf("chat mode = %q, want %q", got, want)
	}
	if got, want := payload.Session.SessionSetup.ProfileID, "profile-1"; got != want {
		t.Fatalf("setup profile_id = %q, want %q", got, want)
	}
	if payload.Session.SessionSetup.Editable {
		t.Fatalf("profile chat should not be editable")
	}
	if got, want := payload.Session.State.Phase, "ready"; got != want {
		t.Fatalf("chat phase = %q, want %q", got, want)
	}

	reset := httptest.NewRequest(http.MethodPost, "/api/chats/chat-1:reset-warm-state", strings.NewReader(`{"actionId":"reset-1"}`))
	reset.Header.Set("Content-Type", "application/json")
	resetRecorder := httptest.NewRecorder()
	mux.ServeHTTP(resetRecorder, reset)
	if resetRecorder.Code != http.StatusOK {
		t.Fatalf("reset status = %d, want 200, body=%s", resetRecorder.Code, resetRecorder.Body.String())
	}
	if got, want := sessions.lastResetSessionID, "chat-1"; got != want {
		t.Fatalf("reset session_id = %q, want %q", got, want)
	}
	if got, want := sessions.lastResetActionID, "reset-1"; got != want {
		t.Fatalf("reset action_id = %q, want %q", got, want)
	}
}

func TestRegisterHandlersCreateChatMainline(t *testing.T) {
	mux := http.NewServeMux()
	sessions := newFakeSessions()
	runs := &fakeRuns{}
	actions := &fakeActions{}
	RegisterHandlers(mux, sessions, sessions, actions, runs, nil, nil)

	request := httptest.NewRequest(http.MethodPost, "/api/chats", strings.NewReader(`{"displayName":"Profile chat","sessionSetup":{"mode":"profile","profileId":"profile-1"}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		ID      string `json:"id"`
		Name    string `json:"displayName"`
		Session struct {
			SessionSetup struct {
				Mode      string `json:"mode"`
				ProfileID string `json:"profileId"`
				Editable  bool   `json:"editable"`
			} `json:"sessionSetup"`
		} `json:"session"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Session.SessionSetup.Mode != "profile" {
		t.Fatalf("chat mode = %q, want profile", payload.Session.SessionSetup.Mode)
	}
	if payload.Name != "Profile chat" {
		t.Fatalf("displayName = %q, want Profile chat", payload.Name)
	}
	if payload.Session.SessionSetup.ProfileID != "profile-1" {
		t.Fatalf("setup profile_id = %q, want profile-1", payload.Session.SessionSetup.ProfileID)
	}
	if payload.Session.SessionSetup.Editable {
		t.Fatalf("profile chat should not be editable")
	}
	if payload.ID == "" {
		t.Fatalf("session id should be generated by server")
	}
	if !strings.HasPrefix(payload.ID, "chat-") {
		t.Fatalf("session id should start with chat-, got %q", payload.ID)
	}
	if got, want := sessions.lastCreateSessionID, payload.ID; got != want {
		t.Fatalf("create session_id = %q, want %q", got, want)
	}
	if got, want := sessions.lastCreateWorkspaceID, payload.ID+"-workspace"; got != want {
		t.Fatalf("create workspace_id = %q, want %q", got, want)
	}
	if got, want := sessions.lastCreateHomeStateID, payload.ID+"-home"; got != want {
		t.Fatalf("create home_state_id = %q, want %q", got, want)
	}

	rename := httptest.NewRequest(http.MethodPost, "/api/chats/"+payload.ID+":rename", strings.NewReader(`{"displayName":"Renamed chat"}`))
	rename.Header.Set("Content-Type", "application/json")
	renameRecorder := httptest.NewRecorder()
	mux.ServeHTTP(renameRecorder, rename)
	if renameRecorder.Code != http.StatusOK {
		t.Fatalf("rename status = %d, want 200, body=%s", renameRecorder.Code, renameRecorder.Body.String())
	}
	var renamedPayload struct {
		Name string `json:"displayName"`
	}
	if err := json.Unmarshal(renameRecorder.Body.Bytes(), &renamedPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if renamedPayload.Name != "Renamed chat" {
		t.Fatalf("renamed displayName = %q, want Renamed chat", renamedPayload.Name)
	}
	if got, want := sessions.lastUpdateSessionID, ""; got != want {
		t.Fatalf("rename should not update session, last update session_id = %q", got)
	}

	list := httptest.NewRequest(http.MethodGet, "/api/chats", nil)
	listRecorder := httptest.NewRecorder()
	mux.ServeHTTP(listRecorder, list)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200, body=%s", listRecorder.Code, listRecorder.Body.String())
	}
	var listPayload struct {
		Items []struct {
			ID   string `json:"id"`
			Name string `json:"displayName"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(listPayload.Items) != 1 || listPayload.Items[0].ID != payload.ID {
		t.Fatalf("list items = %#v, want one created chat", listPayload.Items)
	}
	if listPayload.Items[0].Name != "Renamed chat" {
		t.Fatalf("list displayName = %q, want Renamed chat", listPayload.Items[0].Name)
	}
}

func TestRegisterHandlersInlineChatUpdate(t *testing.T) {
	mux := http.NewServeMux()
	sessions := newFakeSessions()
	sessions.stateByID["chat-2"] = fakeSessionState(&agentsessionv1.AgentSessionSpec{
		SessionId:      "chat-2",
		ProviderId:     "codex",
		ExecutionClass: "cli-standard",
		RuntimeConfig: &agentsessionv1.AgentSessionRuntimeConfig{
			ProviderRuntimeRef: &providerv1.ProviderRuntimeRef{SurfaceId: "primary-1"},
		},
		ResourceConfig: &capv1.AgentResources{
			SnapshotId: "old",
			Instructions: []*capv1.InstructionResource{
				{Kind: capv1.InstructionKind_INSTRUCTION_KIND_RULE, Name: "rule", Content: "content"},
			},
		},
		WorkspaceRef: &agentsessionv1.AgentSessionWorkspaceRef{WorkspaceId: "chat-2-workspace"},
		HomeStateRef: &agentsessionv1.AgentSessionHomeStateRef{HomeStateId: "chat-2-home"},
	})
	RegisterHandlers(mux, sessions, sessions, &fakeActions{}, &fakeRuns{}, nil, nil)

	put := httptest.NewRequest(http.MethodPut, "/api/chats/chat-2", strings.NewReader(`{
		"sessionSetup":{
			"mode":"inline",
			"inline":{
				"runtimeConfig":{
					"providerRuntimeRef":{"surfaceId":"primary-2"},
					"primaryModelSelector":{"providerModelId":"gpt-4.1"},
					"fallbacks":[{"providerRuntimeRef":{"surfaceId":"fallback-1"},"providerModelId":"gpt-4.1-mini"}]
				}
			}
		}
	}`))
	put.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, put)
	if recorder.Code != http.StatusOK {
		t.Fatalf("inline put status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if got, want := sessions.lastUpdateSessionID, "chat-2"; got != want {
		t.Fatalf("update session_id = %q, want %q", got, want)
	}
	if got, want := sessions.lastUpdateProviderID, "codex"; got != want {
		t.Fatalf("update provider_id = %q, want %q", got, want)
	}
	if got, want := sessions.lastUpdateExecutionClass, "cli-standard"; got != want {
		t.Fatalf("update execution_class = %q, want %q", got, want)
	}
	if got := sessions.lastUpdateRuntimeConfig.GetProviderRuntimeRef().GetSurfaceId(); got != "primary-2" {
		t.Fatalf("update runtime primary = %q, want primary-2", got)
	}
	if got := sessions.lastUpdateResourceConfig.GetSnapshotId(); got == "" {
		t.Fatalf("expected normalized resource snapshot id")
	}
}

func TestRegisterHandlersInlineChatUpdateUsesSessionRepoSpec(t *testing.T) {
	mux := http.NewServeMux()
	sessions := newFakeSessions()
	sessions.stateByID["chat-stale"] = fakeSessionState(&agentsessionv1.AgentSessionSpec{
		SessionId:      "chat-stale",
		ProviderId:     "codex",
		ExecutionClass: "cli-standard",
		RuntimeConfig: &agentsessionv1.AgentSessionRuntimeConfig{
			ProviderRuntimeRef: &providerv1.ProviderRuntimeRef{SurfaceId: "primary-1"},
		},
		ResourceConfig: &capv1.AgentResources{
			Instructions: []*capv1.InstructionResource{
				{Kind: capv1.InstructionKind_INSTRUCTION_KIND_RULE, Name: "rule", Content: "content"},
			},
		},
		WorkspaceRef: &agentsessionv1.AgentSessionWorkspaceRef{WorkspaceId: "chat-stale-workspace"},
		HomeStateRef: &agentsessionv1.AgentSessionHomeStateRef{HomeStateId: "chat-stale-home"},
	})
	RegisterHandlers(mux, sessions, sessions, &fakeActions{}, &fakeRuns{}, nil, nil)

	put := httptest.NewRequest(http.MethodPut, "/api/chats/chat-stale", strings.NewReader(`{
		"sessionSetup":{
			"mode":"inline",
			"inline":{
				"runtimeConfig":{"providerRuntimeRef":{"surfaceId":"primary-2"}},
				"resourceConfig":{"instructions":[{"kind":"INSTRUCTION_KIND_RULE","name":"rule","content":"new"}]}
			}
		}
	}`))
	put.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, put)

	if recorder.Code != http.StatusOK {
		t.Fatalf("inline put status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if got, want := sessions.lastUpdateProviderID, "codex"; got != want {
		t.Fatalf("update provider_id = %q, want %q", got, want)
	}
	if got, want := sessions.lastUpdateRuntimeConfig.GetProviderRuntimeRef().GetSurfaceId(), "primary-2"; got != want {
		t.Fatalf("update runtime primary = %q, want %q", got, want)
	}
}

func TestUpsertInlineChatUpdateUsesBoundSessionID(t *testing.T) {
	current := &chatv1.Chat{
		ChatId:    "chat-1",
		SessionId: "session-1",
		SessionState: fakeSessionState(&agentsessionv1.AgentSessionSpec{
			SessionId:      "session-1",
			ProviderId:     "codex",
			ExecutionClass: "cli-standard",
			RuntimeConfig: &agentsessionv1.AgentSessionRuntimeConfig{
				ProviderRuntimeRef: &providerv1.ProviderRuntimeRef{SurfaceId: "primary-1"},
			},
			WorkspaceRef: &agentsessionv1.AgentSessionWorkspaceRef{WorkspaceId: "session-1-workspace"},
			HomeStateRef: &agentsessionv1.AgentSessionHomeStateRef{HomeStateId: "session-1-home"},
		}),
	}
	chats := &boundChatService{current: current}

	_, err := upsertChat(context.Background(), chats, nil, "chat-1", putChatRequest{
		SessionSetup: putChatSessionSetup{
			Mode: chatModeInline,
			Inline: &putInlineChatRequest{
				RuntimeConfig: json.RawMessage(`{"providerRuntimeRef":{"surfaceId":"primary-2"}}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("upsertChat() error = %v", err)
	}
	if got, want := chats.updated.GetSessionId(), "session-1"; got != want {
		t.Fatalf("updated session_id = %q, want %q", got, want)
	}
	if got, want := chats.updated.GetWorkspaceRef().GetWorkspaceId(), "session-1-workspace"; got != want {
		t.Fatalf("updated workspace_id = %q, want %q", got, want)
	}
}

func TestRegisterHandlersTurnMainline(t *testing.T) {
	mux := http.NewServeMux()
	sessions := newFakeSessions()
	sessions.stateByID["chat-1"] = fakeSessionState(&agentsessionv1.AgentSessionSpec{SessionId: "chat-1", ProfileId: "profile-1"})
	actions := &fakeActions{}
	runs := &fakeRuns{}
	RegisterHandlers(mux, sessions, sessions, actions, runs, nil, nil)

	create := httptest.NewRequest(http.MethodPost, "/api/chats/chat-1/turns", strings.NewReader(`{"turnId":"turn-1","runRequest":{"runId":"run-1"}}`))
	create.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	mux.ServeHTTP(createRecorder, create)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create turn status = %d, want 201, body=%s", createRecorder.Code, createRecorder.Body.String())
	}
	if got, want := sessions.lastCreateTurnSessionID, "chat-1"; got != want {
		t.Fatalf("create turn session_id = %q, want %q", got, want)
	}
	if got, want := sessions.lastCreateTurnActionID, "turn-1"; got != want {
		t.Fatalf("create turn action_id = %q, want %q", got, want)
	}
	if got, want := sessions.lastCreateTurnTurnID, "turn-1"; got != want {
		t.Fatalf("create turn turn_id = %q, want %q", got, want)
	}

	createWithActionID := httptest.NewRequest(http.MethodPost, "/api/chats/chat-1/turns", strings.NewReader(`{"actionId":"turn-action-only","runRequest":{"runId":"run-1"}}`))
	createWithActionID.Header.Set("Content-Type", "application/json")
	createWithActionIDRecorder := httptest.NewRecorder()
	mux.ServeHTTP(createWithActionIDRecorder, createWithActionID)
	if createWithActionIDRecorder.Code != http.StatusCreated {
		t.Fatalf("create turn with actionId status = %d, want 201, body=%s", createWithActionIDRecorder.Code, createWithActionIDRecorder.Body.String())
	}
	if got, want := sessions.lastCreateTurnTurnID, "turn-action-only"; got != want {
		t.Fatalf("create turn with actionId turn_id = %q, want %q", got, want)
	}

	get := httptest.NewRequest(http.MethodGet, "/api/chats/chat-1/turns/turn-1", nil)
	getRecorder := httptest.NewRecorder()
	mux.ServeHTTP(getRecorder, get)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("get turn status = %d, want 200, body=%s", getRecorder.Code, getRecorder.Body.String())
	}

	stop := httptest.NewRequest(http.MethodPost, "/api/chats/chat-1/turns/turn-1:stop", nil)
	stopRecorder := httptest.NewRecorder()
	mux.ServeHTTP(stopRecorder, stop)
	if stopRecorder.Code != http.StatusOK {
		t.Fatalf("stop turn status = %d, want 200, body=%s", stopRecorder.Code, stopRecorder.Body.String())
	}

	retry := httptest.NewRequest(http.MethodPost, "/api/chats/chat-1/turns/turn-1:retry", strings.NewReader(`{"newTurnId":"turn-2"}`))
	retry.Header.Set("Content-Type", "application/json")
	retryRecorder := httptest.NewRecorder()
	mux.ServeHTTP(retryRecorder, retry)
	if retryRecorder.Code != http.StatusOK {
		t.Fatalf("retry turn status = %d, want 200, body=%s", retryRecorder.Code, retryRecorder.Body.String())
	}
	if got, want := actions.lastRetrySourceActionID, "turn-1"; got != want {
		t.Fatalf("retry source turn = %q, want %q", got, want)
	}
	if got, want := actions.lastRetryNewTurnID, "turn-2"; got != want {
		t.Fatalf("retry new turn = %q, want %q", got, want)
	}
}

func TestRegisterHandlersRejectsCrossChatTurn(t *testing.T) {
	mux := http.NewServeMux()
	sessions := newFakeSessions()
	sessions.stateByID["chat-1"] = fakeSessionState(&agentsessionv1.AgentSessionSpec{SessionId: "chat-1", ProfileId: "profile-1"})
	actions := &fakeActions{sessionIDOverride: "chat-2"}
	runs := &fakeRuns{}
	RegisterHandlers(mux, sessions, sessions, actions, runs, nil, nil)

	get := httptest.NewRequest(http.MethodGet, "/api/chats/chat-1/turns/turn-1", nil)
	getRecorder := httptest.NewRecorder()
	mux.ServeHTTP(getRecorder, get)
	if getRecorder.Code != http.StatusNotFound {
		t.Fatalf("get turn status = %d, want 404, body=%s", getRecorder.Code, getRecorder.Body.String())
	}
}

func TestRegisterHandlersRejectsChatRunRoute(t *testing.T) {
	mux := http.NewServeMux()
	sessions := newFakeSessions()
	sessions.stateByID["chat-1"] = fakeSessionState(&agentsessionv1.AgentSessionSpec{SessionId: "chat-1", ProfileId: "profile-1"})
	actions := &fakeActions{}
	runs := &fakeRuns{}
	RegisterHandlers(mux, sessions, sessions, actions, runs, nil, nil)

	get := httptest.NewRequest(http.MethodGet, "/api/chats/chat-1/runs/run-1", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, get)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("get run status = %d, want 404, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRegisterHandlersListChatMessages(t *testing.T) {
	mux := http.NewServeMux()
	sessions := newFakeSessions()
	sessions.stateByID["chat-1"] = fakeSessionState(&agentsessionv1.AgentSessionSpec{SessionId: "chat-1", ProfileId: "profile-1"})
	sessions.messagesByChatID["chat-1"] = []*structpb.Struct{
		mustStruct(map[string]any{"id": "user-turn-1", "role": "user", "content": "hello"}),
	}
	RegisterHandlers(mux, sessions, sessions, &fakeActions{}, &fakeRuns{}, nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/api/chats/chat-1/messages", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("messages status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, fragment := range []string{
		`"messages"`,
		`"id":"user-turn-1"`,
		`"role":"user"`,
		`"content":"hello"`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("response missing %q: %s", fragment, body)
		}
	}
}

func TestRegisterHandlersAGUIRun(t *testing.T) {
	mux := http.NewServeMux()
	sessions := newFakeSessions()
	sessions.stateByID["chat-1"] = fakeSessionState(&agentsessionv1.AgentSessionSpec{
		SessionId:    "chat-1",
		ProfileId:    "profile-1",
		WorkspaceRef: &agentsessionv1.AgentSessionWorkspaceRef{WorkspaceId: "chat-1-workspace"},
		HomeStateRef: &agentsessionv1.AgentSessionHomeStateRef{HomeStateId: "chat-1-home"},
	})
	sessions.messagesByChatID["chat-1"] = []*structpb.Struct{
		mustStruct(map[string]any{"id": "user-old", "role": "user", "content": "previous"}),
	}
	actions := &fakeActions{}
	runs := &fakeRuns{}
	outputs := &fakeRunOutputs{
		eventsByRunID: map[string][]runOutputEvent{
			"run-1": {
				{
					Delta: &runeventv1.RunDeltaEvent{
						SessionId: "chat-1",
						RunId:     "run-1",
						Output:    testRunOutput(1, map[string]any{"type": "TEXT_MESSAGE_START", "messageId": "assistant-message", "role": "assistant"}),
					},
				},
				{
					Delta: &runeventv1.RunDeltaEvent{
						SessionId: "chat-1",
						RunId:     "run-1",
						Output:    testRunOutput(2, map[string]any{"type": "TEXT_MESSAGE_CONTENT", "messageId": "assistant-message", "delta": "hello "}),
					},
				},
				{
					Result: &runeventv1.RunResultEvent{
						SessionId: "chat-1",
						RunId:     "run-1",
						Payload: &runeventv1.RunResultEvent_Output{
							Output: testRunOutput(3, map[string]any{"type": "TOOL_CALL_START", "toolCallId": "tool-1", "toolCallName": "shell"}),
						},
					},
				},
				{
					Result: &runeventv1.RunResultEvent{
						SessionId: "chat-1",
						RunId:     "run-1",
						Payload: &runeventv1.RunResultEvent_Output{
							Output: testRunOutput(4, map[string]any{"type": "TOOL_CALL_ARGS", "toolCallId": "tool-1", "delta": `{"summary":"ls -la"}`}),
						},
					},
				},
				{
					Result: &runeventv1.RunResultEvent{
						SessionId: "chat-1",
						RunId:     "run-1",
						Payload: &runeventv1.RunResultEvent_Output{
							Output: testRunOutput(5, map[string]any{"type": "TOOL_CALL_END", "toolCallId": "tool-1"}),
						},
					},
				},
				{
					Result: &runeventv1.RunResultEvent{
						SessionId: "chat-1",
						RunId:     "run-1",
						Payload: &runeventv1.RunResultEvent_Output{
							Output: testRunOutput(6, map[string]any{"type": "TOOL_CALL_RESULT", "messageId": "tool-message-tool-1", "toolCallId": "tool-1", "role": "tool", "content": `{"summary":"ls -la"}`}),
						},
					},
				},
				{
					Result: &runeventv1.RunResultEvent{
						SessionId: "chat-1",
						RunId:     "run-1",
						Payload: &runeventv1.RunResultEvent_Output{
							Output: testRunOutput(7, map[string]any{"type": "TEXT_MESSAGE_CONTENT", "messageId": "assistant-message", "delta": "from run output"}),
						},
					},
				},
				{
					Result: &runeventv1.RunResultEvent{
						SessionId: "chat-1",
						RunId:     "run-1",
						Payload: &runeventv1.RunResultEvent_Output{
							Output: testRunOutput(8, map[string]any{"type": "TEXT_MESSAGE_END", "messageId": "assistant-message"}),
						},
					},
				},
				{
					Result: &runeventv1.RunResultEvent{
						SessionId: "chat-1",
						RunId:     "run-1",
						Payload: &runeventv1.RunResultEvent_Output{
							Output: testRunOutput(9, map[string]any{
								"type": "CUSTOM",
								"name": "run.llm_usage",
								"value": map[string]any{
									"modelId":             "gpt-5",
									"contextWindowTokens": 128000,
									"usage": map[string]any{
										"inputTokens":           12,
										"outputTokens":          7,
										"cachedInputTokens":     2,
										"reasoningOutputTokens": 3,
									},
								},
							}),
						},
					},
				},
				{
					Result: &runeventv1.RunResultEvent{
						SessionId: "chat-1",
						RunId:     "run-1",
						Payload: &runeventv1.RunResultEvent_Output{
							Output: testRunOutput(10, map[string]any{
								"type": "CUSTOM",
								"name": "run.turn_usage",
								"value": map[string]any{
									"usage": map[string]any{
										"inputTokens":           12,
										"outputTokens":          7,
										"cachedInputTokens":     2,
										"reasoningOutputTokens": 3,
									},
									"counters": map[string]any{
										"requestCount":  1,
										"toolCallCount": 1,
									},
								},
							}),
						},
					},
				},
				{
					Result: &runeventv1.RunResultEvent{
						SessionId: "chat-1",
						RunId:     "run-1",
						Payload: &runeventv1.RunResultEvent_TerminalResult{
							TerminalResult: &resultv1.RunResult{Status: resultv1.RunStatus_RUN_STATUS_COMPLETED},
						},
					},
				},
			},
		},
	}
	RegisterHandlers(mux, sessions, sessions, actions, runs, outputs, nil)

	request := httptest.NewRequest(http.MethodPost, "/api/chats/chat-1/session/ag-ui", strings.NewReader(`{
		"threadId":"chat-1",
		"runId":"run-1",
		"parentRunId":"run-0",
		"messages":[{"id":"user-1","role":"user","content":"hello ag-ui"}]
	}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("ag-ui status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content-type = %q, want text/event-stream", got)
	}
	body := recorder.Body.String()
	for _, fragment := range []string{
		`"type":"MESSAGES_SNAPSHOT"`,
		`"id":"user-old"`,
		`"content":"previous"`,
		`"type":"RUN_STARTED"`,
		`"parentRunId":"run-0"`,
		`"input":{"threadId":"chat-1","runId":"run-1","parentRunId":"run-0"`,
		`"type":"STATE_SNAPSHOT"`,
		`"session":{"id":"chat-1"`,
		`"type":"ACTIVITY_SNAPSHOT"`,
		`"activityType":"TURN"`,
		`"messageId":"turn-activity-`,
		`"type":"CUSTOM"`,
		`"type":"TEXT_MESSAGE_START"`,
		`"role":"assistant"`,
		`"type":"TEXT_MESSAGE_CONTENT"`,
		`"type":"TEXT_MESSAGE_END"`,
		`"type":"TOOL_CALL_START"`,
		`"type":"TOOL_CALL_RESULT"`,
		`"toolCallName":"shell"`,
		`"delta":"{\"summary\":\"ls -la\"}"`,
		`"delta":"hello "`,
		`"delta":"from run output"`,
		`"modelId":"gpt-5"`,
		`"toolCallCount":1`,
		`"type":"RUN_FINISHED"`,
		`"runId":"run-1"`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("response missing %q: %s", fragment, body)
		}
	}
	if strings.TrimSpace(sessions.lastCreateTurnActionID) == "" {
		t.Fatalf("expected generated turn id")
	}
	if strings.Contains(body, "chat.session.state.changed") {
		t.Fatalf("state must use AG-UI STATE_SNAPSHOT, got custom state event: %s", body)
	}
	for _, frame := range strings.Split(body, "\n\n") {
		if !strings.Contains(frame, `"type":"STATE_SNAPSHOT"`) {
			continue
		}
		if strings.Contains(frame, `"turn"`) {
			t.Fatalf("turn progress must use AG-UI ACTIVITY_SNAPSHOT, got turn in state: %s", frame)
		}
	}
}

func TestRegisterHandlersAGUIRunRejectsInvalidMessage(t *testing.T) {
	mux := http.NewServeMux()
	sessions := newFakeSessions()
	RegisterHandlers(mux, sessions, sessions, &fakeActions{}, &fakeRuns{}, nil, nil)

	request := httptest.NewRequest(http.MethodPost, "/api/chats/chat-1/session/ag-ui", strings.NewReader(`{
		"threadId":"chat-1",
		"runId":"run-1",
		"messages":[{"role":"user","content":"hello"}]
	}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("ag-ui status = %d, want 400, body=%s", recorder.Code, recorder.Body.String())
	}
	if body := recorder.Body.String(); !strings.Contains(body, "invalid_ag_ui_input") {
		t.Fatalf("response should explain invalid AG-UI input: %s", body)
	}
}

type boundChatService struct {
	current *chatv1.Chat
	updated *agentsessionv1.AgentSessionSpec
}

func (s *boundChatService) GetChat(_ context.Context, chatID string) (*chatv1.Chat, error) {
	if s.current == nil || s.current.GetChatId() != chatID {
		return nil, status.Error(codes.NotFound, "chat not found")
	}
	return cloneChat(s.current), nil
}

func (s *boundChatService) CreateChat(_ context.Context, _ string, _ string, _ string, _ *agentsessionv1.AgentSessionSpec) (*chatv1.Chat, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *boundChatService) ListChats(_ context.Context, _ string, _ int32, _ string) ([]*chatv1.Chat, string, error) {
	return nil, "", status.Error(codes.Unimplemented, "not implemented")
}

func (s *boundChatService) ListChatMessages(context.Context, string, int32, string) ([]*structpb.Struct, string, error) {
	return nil, "", nil
}

func (s *boundChatService) UpdateChatSessionSetup(_ context.Context, chatID string, session *agentsessionv1.AgentSessionSpec) (*chatv1.Chat, error) {
	if s.current == nil || s.current.GetChatId() != chatID {
		return nil, status.Error(codes.NotFound, "chat not found")
	}
	s.updated = proto.Clone(session).(*agentsessionv1.AgentSessionSpec)
	next := cloneChat(s.current)
	next.SessionState = fakeSessionState(session)
	return next, nil
}

func (s *boundChatService) RenameChat(context.Context, string, string) (*chatv1.Chat, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

type fakeSessions struct {
	stateByID                map[string]*agentsessionv1.AgentSessionState
	chatDisplayNameByID      map[string]string
	messagesByChatID         map[string][]*structpb.Struct
	lastResetSessionID       string
	lastResetActionID        string
	lastGetSessionID         string
	createCount              int
	lastCreateSessionID      string
	lastCreateProfileID      string
	lastCreateWorkspaceID    string
	lastCreateHomeStateID    string
	lastUpdateSessionID      string
	lastUpdateProviderID     string
	lastUpdateExecutionClass string
	lastUpdateRuntimeConfig  *agentsessionv1.AgentSessionRuntimeConfig
	lastUpdateResourceConfig *capv1.AgentResources
	lastCreateTurnSessionID  string
	lastCreateTurnActionID   string
	lastCreateTurnTurnID     string
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{
		stateByID:           map[string]*agentsessionv1.AgentSessionState{},
		chatDisplayNameByID: map[string]string{},
		messagesByChatID:    map[string][]*structpb.Struct{},
	}
}

func (s *fakeSessions) Get(_ context.Context, sessionID string) (*agentsessionv1.AgentSessionState, error) {
	s.lastGetSessionID = sessionID
	session, ok := s.stateByID[sessionID]
	if !ok {
		return nil, status.Error(codes.NotFound, "chat not found")
	}
	return proto.Clone(session).(*agentsessionv1.AgentSessionState), nil
}

func (s *fakeSessions) GetChat(_ context.Context, chatID string) (*chatv1.Chat, error) {
	session, ok := s.stateByID[chatID]
	if !ok {
		return nil, status.Error(codes.NotFound, "chat not found")
	}
	chat := fakeChat(chatID, session)
	chat.DisplayName = s.chatDisplayNameByID[chatID]
	return chat, nil
}

func (s *fakeSessions) ListChats(_ context.Context, _ string, _ int32, _ string) ([]*chatv1.Chat, string, error) {
	ids := make([]string, 0, len(s.stateByID))
	for id := range s.stateByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]*chatv1.Chat, 0, len(ids))
	for _, id := range ids {
		chat, err := s.GetChat(context.Background(), id)
		if err != nil {
			return nil, "", err
		}
		items = append(items, chat)
	}
	return items, "", nil
}

func (s *fakeSessions) ListChatMessages(_ context.Context, chatID string, _ int32, _ string) ([]*structpb.Struct, string, error) {
	return append([]*structpb.Struct(nil), s.messagesByChatID[chatID]...), "", nil
}

func (s *fakeSessions) Create(_ context.Context, session *agentsessionv1.AgentSessionSpec) (*agentsessionv1.AgentSessionState, error) {
	s.createCount++
	s.lastCreateSessionID = session.GetSessionId()
	s.lastCreateProfileID = session.GetProfileId()
	s.lastCreateWorkspaceID = session.GetWorkspaceRef().GetWorkspaceId()
	s.lastCreateHomeStateID = session.GetHomeStateRef().GetHomeStateId()
	state := fakeSessionState(session)
	s.stateByID[session.GetSessionId()] = state
	return proto.Clone(state).(*agentsessionv1.AgentSessionState), nil
}

func (s *fakeSessions) CreateChat(ctx context.Context, chatID string, _ string, displayName string, session *agentsessionv1.AgentSessionSpec) (*chatv1.Chat, error) {
	session.SessionId = chatID
	created, err := s.Create(ctx, session)
	if err != nil {
		return nil, err
	}
	chat := fakeChat(chatID, created)
	chat.DisplayName = strings.TrimSpace(displayName)
	s.chatDisplayNameByID[chatID] = chat.GetDisplayName()
	return chat, nil
}

func (s *fakeSessions) Update(_ context.Context, sessionID string, session *agentsessionv1.AgentSessionSpec) (*agentsessionv1.AgentSessionState, error) {
	s.lastUpdateSessionID = sessionID
	s.lastUpdateProviderID = session.GetProviderId()
	s.lastUpdateExecutionClass = session.GetExecutionClass()
	if session.GetRuntimeConfig() != nil {
		s.lastUpdateRuntimeConfig = proto.Clone(session.GetRuntimeConfig()).(*agentsessionv1.AgentSessionRuntimeConfig)
	}
	if session.GetResourceConfig() != nil {
		s.lastUpdateResourceConfig = proto.Clone(session.GetResourceConfig()).(*capv1.AgentResources)
	}
	state := fakeSessionState(session)
	s.stateByID[sessionID] = state
	return proto.Clone(state).(*agentsessionv1.AgentSessionState), nil
}

func (s *fakeSessions) UpdateStatus(_ context.Context, sessionID string, current *agentsessionv1.AgentSessionStatus) (*agentsessionv1.AgentSessionState, error) {
	state, ok := s.stateByID[sessionID]
	if !ok {
		return nil, status.Error(codes.NotFound, "session not found")
	}
	next := proto.Clone(state).(*agentsessionv1.AgentSessionState)
	next.Status = proto.Clone(current).(*agentsessionv1.AgentSessionStatus)
	s.stateByID[sessionID] = next
	return proto.Clone(next).(*agentsessionv1.AgentSessionState), nil
}

func (s *fakeSessions) UpdateChatSessionSetup(ctx context.Context, chatID string, session *agentsessionv1.AgentSessionSpec) (*chatv1.Chat, error) {
	updated, err := s.Update(ctx, chatID, session)
	if err != nil {
		return nil, err
	}
	chat := fakeChat(chatID, updated)
	chat.DisplayName = s.chatDisplayNameByID[chatID]
	return chat, nil
}

func (s *fakeSessions) RenameChat(_ context.Context, chatID string, displayName string) (*chatv1.Chat, error) {
	chat, err := s.GetChat(context.Background(), chatID)
	if err != nil {
		return nil, err
	}
	chat.DisplayName = strings.TrimSpace(displayName)
	s.chatDisplayNameByID[chatID] = chat.GetDisplayName()
	return chat, nil
}

func (s *fakeSessions) CreateTurn(_ context.Context, sessionID string, actionID string, turnID string, runRequest *agentcorev1.RunRequest) (*managementv1.CreateAgentSessionActionResponse, error) {
	s.lastCreateTurnSessionID = sessionID
	s.lastCreateTurnActionID = actionID
	s.lastCreateTurnTurnID = turnID
	return &managementv1.CreateAgentSessionActionResponse{Action: &agentsessionactionv1.AgentSessionActionState{
		Spec: &agentsessionactionv1.AgentSessionActionSpec{
			ActionId:  actionID,
			SessionId: sessionID,
			TurnId:    turnID,
			InputSnapshot: &agentsessionactionv1.AgentSessionActionInputSnapshot{
				Snapshot: &agentsessionactionv1.AgentSessionActionInputSnapshot_RunTurn{
					RunTurn: &agentsessionactionv1.AgentSessionRunTurnSnapshot{RunRequest: runRequest},
				},
			},
		},
		Status: &agentsessionactionv1.AgentSessionActionStatus{
			ActionId: actionID,
			Phase:    agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_SUCCEEDED,
			Message:  "Turn completed successfully.",
			Run:      &agentsessionactionv1.AgentSessionActionRunRef{RunId: runRequest.GetRunId()},
			View: &agentsessionactionv1.AgentSessionActionView{
				DisplayPhase: agentsessionactionv1.AgentSessionActionDisplayPhase_AGENT_SESSION_ACTION_DISPLAY_PHASE_SUCCEEDED,
				CanRetry:     true,
			},
			AttemptCount: 1,
		},
	}}, nil
}

func fakeChat(chatID string, session *agentsessionv1.AgentSessionState) *chatv1.Chat {
	cloned := proto.Clone(session).(*agentsessionv1.AgentSessionState)
	return &chatv1.Chat{
		ChatId:       chatID,
		ScopeId:      defaultChatScopeID,
		SessionId:    cloned.GetSpec().GetSessionId(),
		SessionState: cloned,
	}
}

func (s *fakeSessions) ResetWarmState(_ context.Context, sessionID string, actionID string) (*managementv1.ResetAgentSessionWarmStateResponse, error) {
	s.lastResetSessionID = sessionID
	s.lastResetActionID = actionID
	return &managementv1.ResetAgentSessionWarmStateResponse{
		Action: &agentsessionactionv1.AgentSessionActionState{
			Spec: &agentsessionactionv1.AgentSessionActionSpec{ActionId: actionID, SessionId: sessionID},
		},
	}, nil
}

type fakeActions struct {
	lastRetrySourceActionID string
	lastRetryNewTurnID      string
	sessionIDOverride       string
}

func (a *fakeActions) Get(_ context.Context, actionID string) (*managementv1.GetAgentSessionActionResponse, error) {
	sessionID := "chat-1"
	if strings.TrimSpace(a.sessionIDOverride) != "" {
		sessionID = a.sessionIDOverride
	}
	return &managementv1.GetAgentSessionActionResponse{Action: &agentsessionactionv1.AgentSessionActionState{
		Spec: &agentsessionactionv1.AgentSessionActionSpec{
			ActionId:  actionID,
			SessionId: sessionID,
			TurnId:    actionID,
		},
		Status: &agentsessionactionv1.AgentSessionActionStatus{
			ActionId: actionID,
			Phase:    agentsessionactionv1.AgentSessionActionPhase_AGENT_SESSION_ACTION_PHASE_SUCCEEDED,
			Message:  "Turn completed successfully.",
			Run:      &agentsessionactionv1.AgentSessionActionRunRef{RunId: "run-1"},
			View: &agentsessionactionv1.AgentSessionActionView{
				DisplayPhase: agentsessionactionv1.AgentSessionActionDisplayPhase_AGENT_SESSION_ACTION_DISPLAY_PHASE_SUCCEEDED,
				CanRetry:     true,
			},
			AttemptCount: 1,
		},
	}}, nil
}

func (a *fakeActions) Stop(_ context.Context, actionID string) (*managementv1.StopAgentSessionActionResponse, error) {
	return &managementv1.StopAgentSessionActionResponse{Action: &agentsessionactionv1.AgentSessionActionState{
		Spec: &agentsessionactionv1.AgentSessionActionSpec{
			ActionId:  actionID,
			SessionId: "chat-1",
		},
	}}, nil
}

func (a *fakeActions) Retry(_ context.Context, sourceActionID string, newTurnID string) (*managementv1.RetryAgentSessionActionResponse, error) {
	a.lastRetrySourceActionID = sourceActionID
	a.lastRetryNewTurnID = newTurnID
	return &managementv1.RetryAgentSessionActionResponse{Action: &agentsessionactionv1.AgentSessionActionState{
		Spec: &agentsessionactionv1.AgentSessionActionSpec{
			ActionId:  newTurnID,
			SessionId: "chat-1",
			TurnId:    newTurnID,
		},
	}}, nil
}

type fakeRuns struct{}

func (r *fakeRuns) Get(_ context.Context, runID string) (*managementv1.GetAgentRunResponse, error) {
	return &managementv1.GetAgentRunResponse{Run: &agentrunv1.AgentRunState{
		Spec: &agentrunv1.AgentRunSpec{
			RunId:     runID,
			SessionId: "chat-1",
		},
		Status: &agentrunv1.AgentRunStatus{
			RunId:   runID,
			Phase:   agentrunv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED,
			Message: "AgentRun completed successfully.",
		},
	}}, nil
}

type fakeRunOutputs struct {
	eventsByRunID map[string][]runOutputEvent
}

func (f *fakeRunOutputs) Stream(_ context.Context, runID string, afterSequence uint64, yield func(runOutputEvent) error) error {
	for _, event := range f.eventsByRunID[runID] {
		if sequence := runOutputSequence(event); sequence != 0 && sequence <= afterSequence {
			continue
		}
		cloned := runOutputEvent{}
		if event.Delta != nil {
			cloned.Delta = proto.Clone(event.Delta).(*runeventv1.RunDeltaEvent)
		}
		if event.Result != nil {
			cloned.Result = proto.Clone(event.Result).(*runeventv1.RunResultEvent)
		}
		if err := yield(cloned); err != nil {
			return err
		}
	}
	return nil
}

func fakeSessionState(spec *agentsessionv1.AgentSessionSpec) *agentsessionv1.AgentSessionState {
	cloned := proto.Clone(spec).(*agentsessionv1.AgentSessionSpec)
	if cloned.GetProviderId() == "" {
		cloned.ProviderId = "codex"
	}
	if cloned.GetExecutionClass() == "" && cloned.GetProfileId() == "" {
		cloned.ExecutionClass = "cli-standard"
	}
	return &agentsessionv1.AgentSessionState{
		Spec: cloned,
		Status: &agentsessionv1.AgentSessionStatus{
			SessionId: cloned.GetSessionId(),
			Phase:     agentsessionv1.AgentSessionPhase_AGENT_SESSION_PHASE_READY,
			Message:   "Session ready.",
		},
	}
}

func mustStruct(value map[string]any) *structpb.Struct {
	payload, err := structpb.NewStruct(value)
	if err != nil {
		panic(err)
	}
	return payload
}
