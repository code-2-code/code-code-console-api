package oauthsessions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	oauthv1 "code-code.internal/go-contract/platform/oauth/v1"
)

func TestRegisterHandlersStartRouteUsesOAuthNamespace(t *testing.T) {
	service := &sessionServiceStub{session: oauthTestSession("session-1")}
	mux := http.NewServeMux()
	RegisterHandlers(mux, service)

	request := httptest.NewRequest(http.MethodPost, "/api/oauth/sessions", strings.NewReader(`{"cliId":"codex","flow":"O_AUTH_AUTHORIZATION_FLOW_CODE","targetDisplayName":"Codex"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body.String())
	}
	if service.startRequest.GetCliId() != "codex" {
		t.Fatalf("CliId = %q, want codex", service.startRequest.GetCliId())
	}
	if service.startRequest.GetFlow() != credentialv1.OAuthAuthorizationFlow_O_AUTH_AUTHORIZATION_FLOW_CODE {
		t.Fatalf("Flow = %s, want code", service.startRequest.GetFlow())
	}
}

func TestRegisterHandlersRecordCallbackRoute(t *testing.T) {
	service := &sessionServiceStub{
		session:           oauthTestSession("session-1"),
		recordedSessionID: "session-1",
	}
	mux := http.NewServeMux()
	RegisterHandlers(mux, service)

	request := httptest.NewRequest(http.MethodPost, "/api/oauth/sessions/session-1/callback", strings.NewReader(`{"providerId":"codex","providerRedirectUri":"http://localhost:1455/auth/callback","code":"code-1","state":"state-1"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if service.callbackRequest.GetProviderId() != "codex" {
		t.Fatalf("ProviderId = %q, want codex", service.callbackRequest.GetProviderId())
	}
	if service.getSessionID != "session-1" {
		t.Fatalf("Get session id = %q, want session-1", service.getSessionID)
	}
	var payload struct {
		Spec struct {
			SessionID string `json:"sessionId"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(callback) error = %v", err)
	}
	if payload.Spec.SessionID != "session-1" {
		t.Fatalf("response session id = %q, want session-1", payload.Spec.SessionID)
	}
}

func TestRegisterHandlersRejectsLegacyOAuthSessionsRoute(t *testing.T) {
	service := &sessionServiceStub{session: oauthTestSession("session-1")}
	mux := http.NewServeMux()
	RegisterHandlers(mux, service)

	request := httptest.NewRequest(http.MethodPost, "/api/oauth-sessions/session-1/callback", strings.NewReader(`{}`))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	if service.callbackRequest != nil {
		t.Fatalf("legacy route recorded callback")
	}
}

func oauthTestSession(sessionID string) *credentialv1.OAuthAuthorizationSessionState {
	return &credentialv1.OAuthAuthorizationSessionState{
		Spec: &credentialv1.OAuthAuthorizationSessionSpec{
			SessionId: sessionID,
			CliId:     "codex",
			Flow:      credentialv1.OAuthAuthorizationFlow_O_AUTH_AUTHORIZATION_FLOW_CODE,
		},
		Status: &credentialv1.OAuthAuthorizationSessionStatus{
			Phase: credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_PROCESSING,
		},
	}
}

type sessionServiceStub struct {
	session           *credentialv1.OAuthAuthorizationSessionState
	startRequest      *oauthv1.StartOAuthAuthorizationSessionRequest
	callbackRequest   *oauthv1.RecordOAuthCodeCallbackRequest
	recordedSessionID string
	getSessionID      string
	cancelSessionID   string
}

func (s *sessionServiceStub) Start(_ context.Context, request *oauthv1.StartOAuthAuthorizationSessionRequest) (*credentialv1.OAuthAuthorizationSessionState, error) {
	s.startRequest = request
	return s.session, nil
}

func (s *sessionServiceStub) Get(_ context.Context, sessionID string) (*credentialv1.OAuthAuthorizationSessionState, error) {
	s.getSessionID = sessionID
	return s.session, nil
}

func (s *sessionServiceStub) Cancel(_ context.Context, sessionID string) (*credentialv1.OAuthAuthorizationSessionState, error) {
	s.cancelSessionID = sessionID
	return s.session, nil
}

func (s *sessionServiceStub) RecordCodeCallback(_ context.Context, request *oauthv1.RecordOAuthCodeCallbackRequest) (string, error) {
	s.callbackRequest = request
	return s.recordedSessionID, nil
}
