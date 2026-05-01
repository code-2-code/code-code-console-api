package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"code-code.internal/console-api/internal/platformclient"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
)

func newTestService() providerManagementStub {
	return providerManagementStub{}
}

func TestRegisterHandlersListRoutes(t *testing.T) {
	service := newTestService()

	mux := http.NewServeMux()
	RegisterHandlers(mux, service)

	surfacesRequest := httptest.NewRequest(http.MethodGet, "/api/providers/surfaces", nil)
	surfacesRecorder := httptest.NewRecorder()
	mux.ServeHTTP(surfacesRecorder, surfacesRequest)

	if surfacesRecorder.Code != http.StatusOK {
		t.Fatalf("surfaces status = %d, want 200", surfacesRecorder.Code)
	}
	var surfacesPayload struct {
		Items []struct {
			SurfaceID   string `json:"surfaceId"`
			DisplayName string `json:"displayName"`
		} `json:"items"`
	}
	if err := json.Unmarshal(surfacesRecorder.Body.Bytes(), &surfacesPayload); err != nil {
		t.Fatalf("json.Unmarshal(surfaces) error = %v", err)
	}
	if len(surfacesPayload.Items) != 1 || surfacesPayload.Items[0].SurfaceID != "openai-compatible" {
		t.Fatalf("surfaces payload = %#v", surfacesPayload.Items)
	}

	providersRequest := httptest.NewRequest(http.MethodGet, "/api/providers", nil)
	providersRecorder := httptest.NewRecorder()
	mux.ServeHTTP(providersRecorder, providersRequest)

	if providersRecorder.Code != http.StatusOK {
		t.Fatalf("providers status = %d, want 200", providersRecorder.Code)
	}
}

func TestRegisterHandlersConnectRoute(t *testing.T) {
	service := newTestService()

	mux := http.NewServeMux()
	RegisterHandlers(mux, service)

	request := httptest.NewRequest(http.MethodPost, "/api/providers/connect", strings.NewReader(`{"addMethod":"PROVIDER_ADD_METHOD_API_KEY","vendorId":"openai","displayName":"OpenAI","apiKey":{"apiKey":"sk-test"}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Provider struct {
			ProviderID  string `json:"providerId"`
			DisplayName string `json:"displayName"`
		} `json:"provider"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(connect) error = %v", err)
	}
	if payload.Provider.ProviderID != "provider-provider-openai" || payload.Provider.DisplayName != "OpenAI" {
		t.Fatalf("connect payload = %#v", payload.Provider)
	}
}

func TestRegisterHandlersGetConnectSessionRoute(t *testing.T) {
	service := newTestService()

	mux := http.NewServeMux()
	RegisterHandlers(mux, service)

	request := httptest.NewRequest(http.MethodGet, "/api/providers/connect/sessions/session-1", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Session struct {
			SessionID string `json:"sessionId"`
			Phase     string `json:"phase"`
		} `json:"session"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(session) error = %v", err)
	}
	if payload.Session.SessionID != "session-1" || payload.Session.Phase != "PROVIDER_CONNECT_SESSION_PHASE_AWAITING_USER" {
		t.Fatalf("session payload = %#v", payload.Session)
	}
}

func TestRegisterHandlersUpdateProviderObservabilityAuthenticationRoute(t *testing.T) {
	service := newTestService()

	mux := http.NewServeMux()
	RegisterHandlers(mux, service)

	request := httptest.NewRequest(http.MethodPost, "/api/providers/provider-1/observability-authentication", strings.NewReader(`{"token":"session-test"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		ProviderID  string `json:"providerId"`
		DisplayName string `json:"displayName"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(provider) error = %v", err)
	}
	if payload.ProviderID != "provider-1" || payload.DisplayName != "OpenAI" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestRegisterHandlersGetProviderAuthenticationSummaryRoute(t *testing.T) {
	service := newTestService()

	mux := http.NewServeMux()
	RegisterHandlers(mux, service)

	request := httptest.NewRequest(http.MethodGet, "/api/providers/provider-1/authentication-summary", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"fieldId":"project_id"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"value":"projects/123"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}

func TestRegisterHandlersProbeProviderModelCatalogRoute(t *testing.T) {
	service := newTestService()

	mux := http.NewServeMux()
	RegisterHandlers(mux, service)

	request := httptest.NewRequest(http.MethodPost, "/api/providers/provider-1/model-catalog:probe", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"providerId":"provider-1"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"message":"provider model catalog probe completed"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}

type providerManagementStub struct{}

func (providerManagementStub) ListProviderSurfaceMetadata(context.Context) ([]*supportv1.Surface, error) {
	return []*supportv1.Surface{{
		SurfaceId:     "openai-compatible",
		ProductInfoId: "openai",
	}}, nil
}

func (providerManagementStub) ListProviders(context.Context) ([]*managementv1.ProviderView, error) {
	return []*managementv1.ProviderView{{ProviderId: "provider-provider-openai", DisplayName: "OpenAI"}}, nil
}

func (providerManagementStub) UpdateProvider(_ context.Context, providerID string, request *managementv1.UpdateProviderRequest) (*managementv1.ProviderView, error) {
	return &managementv1.ProviderView{ProviderId: providerID, DisplayName: request.GetProvider().GetDisplayName()}, nil
}

func (providerManagementStub) UpdateProviderAuthentication(_ context.Context, providerID string, _ *managementv1.UpdateProviderAuthenticationRequest) (*managementv1.UpdateProviderAuthenticationResponse, error) {
	return &managementv1.UpdateProviderAuthenticationResponse{
		Outcome: &managementv1.UpdateProviderAuthenticationResponse_Provider{
			Provider: &managementv1.ProviderView{ProviderId: providerID, DisplayName: "OpenAI"},
		},
	}, nil
}

func (providerManagementStub) UpdateProviderObservabilityAuthentication(_ context.Context, providerID string, _ *managementv1.UpdateProviderObservabilityAuthenticationRequest) (*managementv1.ProviderView, error) {
	return &managementv1.ProviderView{ProviderId: providerID, DisplayName: "OpenAI"}, nil
}

func (providerManagementStub) GetProviderAuthenticationSummary(_ context.Context, providerID string) (*platformclient.ProviderAuthenticationSummary, error) {
	return &platformclient.ProviderAuthenticationSummary{
		Provider: &platformclient.CredentialSubjectSummary{
			CredentialID: providerID + "-credential",
		},
		Observability: &platformclient.CredentialSubjectSummary{
			CredentialID: providerID + "-observability",
			Fields: []platformclient.CredentialSubjectSummaryField{{
				FieldID: "project_id",
				Label:   "Project ID",
				Value:   "projects/123",
			}},
		},
	}, nil
}

func (providerManagementStub) ProbeProviderModelCatalog(_ context.Context, providerID string) (*managementv1.ProbeProviderModelCatalogResponse, error) {
	return &managementv1.ProbeProviderModelCatalogResponse{
		ProviderId:  providerID,
		ProviderIds: []string{providerID},
		Message:     "provider model catalog probe completed",
	}, nil
}

func (providerManagementStub) DeleteProvider(context.Context, string) error { return nil }

func (providerManagementStub) Connect(_ context.Context, request *managementv1.ConnectProviderRequest) (*managementv1.ConnectProviderResponse, error) {
	return &managementv1.ConnectProviderResponse{
		Outcome: &managementv1.ConnectProviderResponse_Provider{
			Provider: &managementv1.ProviderView{
				ProviderId:  "provider-provider-openai",
				DisplayName: request.GetDisplayName(),
			},
		},
	}, nil
}

func (providerManagementStub) GetConnectSession(_ context.Context, sessionID string) (*managementv1.ProviderConnectSessionView, error) {
	return &managementv1.ProviderConnectSessionView{
		SessionId: sessionID,
		Phase:     providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_AWAITING_USER,
	}, nil
}

func (providerManagementStub) WatchStatusEvents(_ context.Context, _ []string, yield func(*managementv1.ProviderStatusEvent) error) error {
	return yield(&managementv1.ProviderStatusEvent{
		ProviderId: "provider-provider-openai",
		Kind:       providerservicev1.ProviderStatusEventKind_PROVIDER_STATUS_EVENT_KIND_WORKFLOW,
	})
}

func (providerManagementStub) ProbeProvidersObservability(_ context.Context, providerIDs []string) (*managementv1.ProbeProviderObservabilityResponse, error) {
	providerID := firstProviderID(providerIDs)
	return &managementv1.ProbeProviderObservabilityResponse{
		ProviderId: providerID,
		Outcome:    providerservicev1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_EXECUTED,
		Message:    "probe completed",
	}, nil
}

func firstProviderID(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
