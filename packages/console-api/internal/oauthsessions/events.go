package oauthsessions

import (
	"fmt"
	"net/http"
	"time"

	"code-code.internal/console-api/internal/httpjson"
	credentialv1 "code-code.internal/go-contract/credential/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	sessionEventsPollInterval      = 1 * time.Second
	sessionEventsHeartbeatInterval = 15 * time.Second
	sessionEventName               = "session"
)

var sessionProtoJSONMarshaler = protojson.MarshalOptions{EmitUnpopulated: true}

func writeSessionEvents(w http.ResponseWriter, r *http.Request, service sessionService, sessionID string) {
	session, err := service.Get(r.Context(), sessionID)
	if err != nil {
		httpjson.WriteServiceError(w, http.StatusBadRequest, "get_oauth_session_failed", err)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpjson.WriteError(w, http.StatusInternalServerError, "streaming_unsupported", "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	payload, err := sessionEventPayload(session)
	if err != nil {
		return
	}
	if err := writeSSEMessage(w, payload); err != nil {
		return
	}
	flusher.Flush()
	if isTerminalProtoPhase(session.GetStatus().GetPhase()) {
		return
	}

	pollTicker := time.NewTicker(sessionEventsPollInterval)
	heartbeatTicker := time.NewTicker(sessionEventsHeartbeatInterval)
	defer pollTicker.Stop()
	defer heartbeatTicker.Stop()

	lastPayload := payload
	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeatTicker.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-pollTicker.C:
			nextSession, err := service.Get(r.Context(), sessionID)
			if err != nil {
				return
			}
			nextPayload, err := sessionEventPayload(nextSession)
			if err != nil {
				return
			}
			if nextPayload == lastPayload {
				continue
			}
			if err := writeSSEMessage(w, nextPayload); err != nil {
				return
			}
			flusher.Flush()
			lastPayload = nextPayload
			if isTerminalProtoPhase(nextSession.GetStatus().GetPhase()) {
				return
			}
		}
	}
}

func sessionEventPayload(session *credentialv1.OAuthAuthorizationSessionState) (string, error) {
	data, err := sessionProtoJSONMarshaler.Marshal(session)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeSSEMessage(w http.ResponseWriter, payload string) error {
	if _, err := fmt.Fprintf(w, "event: %s\n", sessionEventName); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		return err
	}
	return nil
}

func isTerminalProtoPhase(phase credentialv1.OAuthAuthorizationPhase) bool {
	switch phase {
	case credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_SUCCEEDED,
		credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_FAILED,
		credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_EXPIRED,
		credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_CANCELED:
		return true
	default:
		return false
	}
}
