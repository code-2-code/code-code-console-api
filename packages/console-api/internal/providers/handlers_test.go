package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
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

	surfaceBindingsRequest := httptest.NewRequest(http.MethodGet, "/api/providers/surface-bindings", nil)
	surfaceBindingsRecorder := httptest.NewRecorder()
	mux.ServeHTTP(surfaceBindingsRecorder, surfaceBindingsRequest)

	if surfaceBindingsRecorder.Code != http.StatusOK {
		t.Fatalf("surfaceBindings status = %d, want 200", surfaceBindingsRecorder.Code)
	}
	var surfaceBindingsPayload struct {
		Items []struct {
			SurfaceID   string `json:"surfaceId"`
			DisplayName string `json:"displayName"`
		} `json:"items"`
	}
	if err := json.Unmarshal(surfaceBindingsRecorder.Body.Bytes(), &surfaceBindingsPayload); err != nil {
		t.Fatalf("json.Unmarshal(surfaceBindings) error = %v", err)
	}
	if len(surfaceBindingsPayload.Items) != 1 || surfaceBindingsPayload.Items[0].SurfaceID != "sample-openai-compatible" {
		t.Fatalf("surfaceBindings payload = %#v", surfaceBindingsPayload.Items)
	}
	if surfaceBindingsPayload.Items[0].DisplayName != "默认接入" {
		t.Fatalf("DisplayName = %q, want %q", surfaceBindingsPayload.Items[0].DisplayName, "默认接入")
	}
}

func TestRegisterHandlersCreateAcceptsProtoJSON(t *testing.T) {
	service := newTestService()

	mux := http.NewServeMux()
	RegisterHandlers(mux, service)

	request := httptest.NewRequest(http.MethodPost, "/api/providers/surface-bindings", strings.NewReader(`{"displayName":"test","surfaceId":"openai-compatible"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
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

type providerManagementStub struct{}

func (providerManagementStub) ListProviderSurfaceMetadata(context.Context) ([]*providerv1.ProviderSurface, error) {
	return []*providerv1.ProviderSurface{{
		SurfaceId:   "openai-compatible",
		DisplayName: "OpenAI Compatible",
		Kind:        providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API,
		SupportedCredentialKinds: []credentialv1.CredentialKind{
			credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY,
		},
	}}, nil
}

func (providerManagementStub) ListProviders(context.Context) ([]*managementv1.ProviderView, error) {
	return []*managementv1.ProviderView{{ProviderId: "provider-provider-openai", DisplayName: "OpenAI"}}, nil
}

func (providerManagementStub) ListProviderSurfaceBindings(context.Context) ([]*managementv1.ProviderSurfaceBindingView, error) {
	return []*managementv1.ProviderSurfaceBindingView{{SurfaceId: "sample-openai-compatible", DisplayName: "默认接入"}}, nil
}

func (providerManagementStub) CreateProviderSurfaceBinding(_ context.Context, request *managementv1.UpsertProviderSurfaceBindingRequest) (*managementv1.ProviderSurfaceBindingView, error) {
	return &managementv1.ProviderSurfaceBindingView{SurfaceId: request.GetSurfaceId(), DisplayName: request.GetDisplayName()}, nil
}

func (providerManagementStub) UpdateProviderSurfaceBinding(_ context.Context, instanceID string, request *managementv1.UpsertProviderSurfaceBindingRequest) (*managementv1.ProviderSurfaceBindingView, error) {
	return &managementv1.ProviderSurfaceBindingView{SurfaceId: instanceID, DisplayName: request.GetDisplayName()}, nil
}

func (providerManagementStub) DeleteProviderSurfaceBinding(context.Context, string) error { return nil }

func (providerManagementStub) UpdateProvider(_ context.Context, providerID string, request *managementv1.UpdateProviderRequest) (*managementv1.ProviderView, error) {
	return &managementv1.ProviderView{ProviderId: providerID, DisplayName: request.GetDisplayName()}, nil
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
		Phase:     managementv1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_AWAITING_USER,
	}, nil
}

func (providerManagementStub) WatchStatusEvents(_ context.Context, _ []string, yield func(*managementv1.ProviderStatusEvent) error) error {
	return yield(&managementv1.ProviderStatusEvent{
		ProviderId: "provider-provider-openai",
		Kind:       managementv1.ProviderStatusEventKind_PROVIDER_STATUS_EVENT_KIND_WORKFLOW,
	})
}

func (providerManagementStub) ProbeProvidersObservability(_ context.Context, providerIDs []string) (*managementv1.ProbeProviderObservabilityResponse, error) {
	providerID := firstProviderID(providerIDs)
	return &managementv1.ProbeProviderObservabilityResponse{
		ProviderId: providerID,
		Outcome:    managementv1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_EXECUTED,
		Message:    "probe completed",
	}, nil
}

func firstProviderID(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
