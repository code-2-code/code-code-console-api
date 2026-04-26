package chats

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	agentsessionv1 "code-code.internal/go-contract/platform/agent_session/v1"
)

func TestRegisterHandlersAGUICapabilities(t *testing.T) {
	mux := http.NewServeMux()
	sessions := newFakeSessions()
	sessions.stateByID["chat-1"] = fakeSessionState(&agentsessionv1.AgentSessionSpec{SessionId: "chat-1"})
	sessions.chatDisplayNameByID["chat-1"] = "Design chat"
	RegisterHandlers(mux, sessions, sessions, &fakeActions{}, &fakeRuns{}, nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/api/chats/chat-1/session/ag-ui/capabilities", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("capabilities status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, fragment := range []string{
		`"identity"`,
		`"name":"Design chat"`,
		`"transport":{"streaming":true}`,
		`"tools":{"supported":true}`,
		`"state":{"persistentState":true,"snapshots":true}`,
		`"reasoning":{"streaming":true,"supported":true}`,
		`"activity":{"snapshots":true,"turnSteps":true}`,
		`"serialization":{"messagesSnapshot":true,"runStartedInput":true}`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("capabilities missing %q: %s", fragment, body)
		}
	}
	for _, fragment := range []string{
		`"execution"`,
		`"multimodal"`,
		`"resumable"`,
	} {
		if strings.Contains(body, fragment) {
			t.Fatalf("capabilities should not declare unsupported %q: %s", fragment, body)
		}
	}
}
