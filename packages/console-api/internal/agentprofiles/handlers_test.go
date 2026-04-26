package agentprofiles

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	agentprofilev1 "code-code.internal/go-contract/platform/agent_profile/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

func TestRegisterHandlersListAgentProfiles(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(mux, testService{})

	request := httptest.NewRequest(http.MethodGet, "/api/agent-profiles", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var payload struct {
		Items []struct {
			ProfileID string `json:"profileId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].ProfileID != "general-operator" {
		t.Fatalf("payload = %#v", payload.Items)
	}
}

func TestRegisterHandlersCreateAndDeleteAgentProfile(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(mux, testService{})

	create := httptest.NewRequest(http.MethodPost, "/api/agent-profiles", strings.NewReader(`{"profileId":"new-profile","name":"New Profile","selectionStrategy":{"providerId":"codex","executionClass":"default","fallbacks":[{"providerRuntimeRef":{"instanceId":"openai-main"},"providerModelId":"gpt-5"}]}}`))
	create.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	mux.ServeHTTP(createRecorder, create)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201, body=%s", createRecorder.Code, createRecorder.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/agent-profiles/new-profile", nil)
	deleteRecorder := httptest.NewRecorder()
	mux.ServeHTTP(deleteRecorder, deleteRequest)
	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200, body=%s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

type testService struct{}

func (testService) List(context.Context) ([]*managementv1.AgentProfileListItem, error) {
	return []*managementv1.AgentProfileListItem{{ProfileId: "general-operator", Name: "General Operator"}}, nil
}

func (testService) Get(context.Context, string) (*agentprofilev1.AgentProfile, error) {
	return &agentprofilev1.AgentProfile{ProfileId: "general-operator", Name: "General Operator"}, nil
}

func (testService) Create(_ context.Context, request *managementv1.UpsertAgentProfileRequest) (*agentprofilev1.AgentProfile, error) {
	return &agentprofilev1.AgentProfile{ProfileId: request.GetProfileId(), Name: request.GetName(), SelectionStrategy: request.GetSelectionStrategy()}, nil
}

func (testService) Update(_ context.Context, profileID string, request *managementv1.UpsertAgentProfileRequest) (*agentprofilev1.AgentProfile, error) {
	return &agentprofilev1.AgentProfile{ProfileId: profileID, Name: request.GetName(), SelectionStrategy: request.GetSelectionStrategy()}, nil
}

func (testService) Delete(context.Context, string) error { return nil }
