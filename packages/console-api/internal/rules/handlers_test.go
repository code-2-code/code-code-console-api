package rules

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	rulev1 "code-code.internal/go-contract/platform/rule/v1"
)

func TestRegisterHandlersListRules(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(mux, testService{})

	request := httptest.NewRequest(http.MethodGet, "/api/rules", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var payload struct {
		Items []struct {
			RuleID string `json:"ruleId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].RuleID != "security-review" {
		t.Fatalf("payload = %#v", payload.Items)
	}
}

func TestRegisterHandlersCreateAndDeleteRule(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(mux, testService{})

	create := httptest.NewRequest(http.MethodPost, "/api/rules", strings.NewReader(`{"ruleId":"no-secrets","name":"No Secrets","description":"Protect secrets","content":"Never print secrets."}`))
	create.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	mux.ServeHTTP(createRecorder, create)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201, body=%s", createRecorder.Code, createRecorder.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/rules/no-secrets", nil)
	deleteRecorder := httptest.NewRecorder()
	mux.ServeHTTP(deleteRecorder, deleteRequest)
	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200, body=%s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

type testService struct{}

func (testService) List(context.Context) ([]*managementv1.RuleListItem, error) {
	return []*managementv1.RuleListItem{{RuleId: "security-review", Name: "Security Review"}}, nil
}

func (testService) Get(context.Context, string) (*rulev1.Rule, error) {
	return &rulev1.Rule{RuleId: "security-review", Name: "Security Review"}, nil
}

func (testService) Create(_ context.Context, request *managementv1.UpsertRuleRequest) (*rulev1.Rule, error) {
	return &rulev1.Rule{RuleId: request.GetRuleId(), Name: request.GetName(), Description: request.GetDescription(), Content: request.GetContent()}, nil
}

func (testService) Update(_ context.Context, ruleID string, request *managementv1.UpsertRuleRequest) (*rulev1.Rule, error) {
	return &rulev1.Rule{RuleId: ruleID, Name: request.GetName(), Description: request.GetDescription(), Content: request.GetContent()}, nil
}

func (testService) Delete(context.Context, string) error { return nil }
