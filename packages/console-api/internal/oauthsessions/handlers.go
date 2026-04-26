package oauthsessions

import (
	"context"
	"net/http"
	"strings"

	"code-code.internal/console-api/internal/httpjson"
	credentialv1 "code-code.internal/go-contract/credential/v1"
	oauthv1 "code-code.internal/go-contract/platform/oauth/v1"
)

const (
	oauthSessionsPath      = "/api/oauth/sessions"
	oauthSessionPathPrefix = oauthSessionsPath + "/"
)

type sessionService interface {
	Start(context.Context, *oauthv1.StartOAuthAuthorizationSessionRequest) (*credentialv1.OAuthAuthorizationSessionState, error)
	Get(context.Context, string) (*credentialv1.OAuthAuthorizationSessionState, error)
	Cancel(context.Context, string) (*credentialv1.OAuthAuthorizationSessionState, error)
	RecordCodeCallback(context.Context, *oauthv1.RecordOAuthCodeCallbackRequest) (string, error)
}

// RegisterHandlers registers OAuth session routes onto the provided mux.
func RegisterHandlers(mux *http.ServeMux, service sessionService) {
	mux.HandleFunc(oauthSessionsPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		var request oauthv1.StartOAuthAuthorizationSessionRequest
		if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		session, err := service.Start(r.Context(), &request)
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusBadRequest, "start_oauth_session_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusCreated, session)
	})

	mux.HandleFunc(oauthSessionPathPrefix, func(w http.ResponseWriter, r *http.Request) {
		sessionID, action, ok := parseSessionRoute(r.URL.Path)
		if !ok {
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "oauth session route not found")
			return
		}
		switch action {
		case "":
			handleSession(w, r, service, sessionID)
		case "events":
			if r.Method != http.MethodGet {
				httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
				return
			}
			writeSessionEvents(w, r, service, sessionID)
		case "callback":
			handleCallback(w, r, service, sessionID)
		default:
			httpjson.WriteError(w, http.StatusNotFound, "not_found", "oauth session route not found")
		}
	})
}

func handleSession(w http.ResponseWriter, r *http.Request, service sessionService, sessionID string) {
	switch r.Method {
	case http.MethodGet:
		session, err := service.Get(r.Context(), sessionID)
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusBadRequest, "get_oauth_session_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, session)
	case http.MethodDelete:
		session, err := service.Cancel(r.Context(), sessionID)
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusBadRequest, "cancel_oauth_session_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, session)
	default:
		httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func handleCallback(w http.ResponseWriter, r *http.Request, service sessionService, sessionID string) {
	if r.Method != http.MethodPost {
		httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var request oauthv1.RecordOAuthCodeCallbackRequest
	if err := httpjson.DecodeProtoJSON(r, &request); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	recordedSessionID, err := service.RecordCodeCallback(r.Context(), &request)
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "record_oauth_callback_failed", err)
		return
	}
	if recorded := strings.TrimSpace(recordedSessionID); recorded != "" && recorded != sessionID {
		httpjson.WriteError(w, http.StatusConflict, "oauth_session_mismatch", "oauth callback session mismatch")
		return
	}
	session, err := service.Get(r.Context(), sessionID)
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "get_oauth_session_failed", err)
		return
	}
	httpjson.WriteProtoJSON(w, http.StatusOK, session)
}

func parseSessionRoute(routePath string) (string, string, bool) {
	path := strings.TrimPrefix(routePath, oauthSessionPathPrefix)
	if path == "" {
		return "", "", false
	}
	parts := strings.Split(path, "/")
	if len(parts) > 2 {
		return "", "", false
	}
	sessionID := strings.TrimSpace(parts[0])
	if sessionID == "" {
		return "", "", false
	}
	if len(parts) == 1 {
		return sessionID, "", true
	}
	action := strings.TrimSpace(parts[1])
	if action == "" {
		return "", "", false
	}
	return sessionID, action, true
}
