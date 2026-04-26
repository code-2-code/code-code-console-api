package templates

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

func newTestService() templateManagementStub {
	return templateManagementStub{}
}

func TestRegisterHandlersListTemplates(t *testing.T) {
	service := newTestService()

	mux := http.NewServeMux()
	RegisterHandlers(mux, service)

	request := httptest.NewRequest(http.MethodGet, "/api/templates", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var payload struct {
		Items []struct {
			TemplateID  string `json:"templateId"`
			DisplayName string `json:"displayName"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Items) == 0 {
		t.Fatal("templates payload is empty")
	}
}

func TestRegisterHandlersApplyTemplate(t *testing.T) {
	service := newTestService()
	templateID := "openai-compatible"

	mux := http.NewServeMux()
	RegisterHandlers(mux, service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/templates/"+templateID+"/apply",
		strings.NewReader(`{"namespace":"default","displayName":"Test","providerId":"test-1"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", recorder.Code, recorder.Body.String())
	}
}

type templateManagementStub struct{}

func (templateManagementStub) List(context.Context) ([]*managementv1.TemplateView, error) {
	return []*managementv1.TemplateView{{TemplateId: "openai-compatible", DisplayName: "OpenAI Compatible"}}, nil
}

func (templateManagementStub) Apply(_ context.Context, templateID string, request *managementv1.ApplyTemplateRequest) (*managementv1.ApplyTemplateResult, error) {
	return &managementv1.ApplyTemplateResult{TemplateId: templateID, Namespace: request.GetNamespace(), DisplayName: request.GetDisplayName(), ProviderId: request.GetProviderId()}, nil
}
