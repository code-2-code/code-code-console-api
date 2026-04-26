package platformclient

import (
	"context"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	oauthv1 "code-code.internal/go-contract/platform/oauth/v1"
)

func (o *OAuthSessions) Start(ctx context.Context, request *oauthv1.StartOAuthAuthorizationSessionRequest) (*credentialv1.OAuthAuthorizationSessionState, error) {
	client, err := o.client.requireOAuthSession()
	if err != nil {
		return nil, err
	}
	response, err := client.StartOAuthAuthorizationSession(ctx, request)
	if err != nil {
		return nil, err
	}
	return response.GetSession(), nil
}

func (o *OAuthSessions) Get(ctx context.Context, sessionID string) (*credentialv1.OAuthAuthorizationSessionState, error) {
	client, err := o.client.requireOAuthSession()
	if err != nil {
		return nil, err
	}
	response, err := client.GetOAuthAuthorizationSession(ctx, &oauthv1.GetOAuthAuthorizationSessionRequest{SessionId: sessionID})
	if err != nil {
		return nil, err
	}
	return response.GetSession(), nil
}

func (o *OAuthSessions) Cancel(ctx context.Context, sessionID string) (*credentialv1.OAuthAuthorizationSessionState, error) {
	client, err := o.client.requireOAuthSession()
	if err != nil {
		return nil, err
	}
	response, err := client.CancelOAuthAuthorizationSession(ctx, &oauthv1.CancelOAuthAuthorizationSessionRequest{SessionId: sessionID})
	if err != nil {
		return nil, err
	}
	return response.GetSession(), nil
}

func (o *OAuthSessions) RecordCodeCallback(ctx context.Context, request *oauthv1.RecordOAuthCodeCallbackRequest) (string, error) {
	client, err := o.client.requireOAuthCallback()
	if err != nil {
		return "", err
	}
	response, err := client.RecordOAuthCodeCallback(ctx, request)
	if err != nil {
		return "", err
	}
	return response.GetSessionId(), nil
}
