package platformclient

import (
	"context"

	agentcorev1 "code-code.internal/go-contract/agent/core/v1"
	agentsessionv1 "code-code.internal/go-contract/platform/agent_session/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

func (s *AgentSessions) Get(ctx context.Context, sessionID string) (*agentsessionv1.AgentSessionState, error) {
	client, err := s.client.requireSessionManagement()
	if err != nil {
		return nil, err
	}
	response, err := client.GetAgentSession(ctx, &managementv1.GetAgentSessionRequest{SessionId: sessionID})
	if err != nil {
		return nil, err
	}
	return response.GetSession(), nil
}

func (s *AgentSessions) CreateTurn(ctx context.Context, sessionID string, actionID string, turnID string, runRequest *agentcorev1.RunRequest) (*managementv1.CreateAgentSessionActionResponse, error) {
	client, err := s.client.requireSessionManagement()
	if err != nil {
		return nil, err
	}
	return client.CreateAgentSessionAction(ctx, &managementv1.CreateAgentSessionActionRequest{
		SessionId:  sessionID,
		ActionId:   actionID,
		TurnId:     turnID,
		RunRequest: runRequest,
	})
}

func (s *AgentSessions) ResetWarmState(ctx context.Context, sessionID string, actionID string) (*managementv1.ResetAgentSessionWarmStateResponse, error) {
	client, err := s.client.requireSessionManagement()
	if err != nil {
		return nil, err
	}
	return client.ResetAgentSessionWarmState(ctx, &managementv1.ResetAgentSessionWarmStateRequest{
		SessionId: sessionID,
		ActionId:  actionID,
	})
}

func (a *AgentSessionActions) Get(ctx context.Context, actionID string) (*managementv1.GetAgentSessionActionResponse, error) {
	client, err := a.client.requireSessionManagement()
	if err != nil {
		return nil, err
	}
	return client.GetAgentSessionAction(ctx, &managementv1.GetAgentSessionActionRequest{ActionId: actionID})
}

func (a *AgentSessionActions) Stop(ctx context.Context, actionID string) (*managementv1.StopAgentSessionActionResponse, error) {
	client, err := a.client.requireSessionManagement()
	if err != nil {
		return nil, err
	}
	return client.StopAgentSessionAction(ctx, &managementv1.StopAgentSessionActionRequest{ActionId: actionID})
}

func (a *AgentSessionActions) Retry(ctx context.Context, sourceActionID string, newTurnID string) (*managementv1.RetryAgentSessionActionResponse, error) {
	client, err := a.client.requireSessionManagement()
	if err != nil {
		return nil, err
	}
	return client.RetryAgentSessionAction(ctx, &managementv1.RetryAgentSessionActionRequest{
		SourceActionId: sourceActionID,
		NewTurnId:      newTurnID,
	})
}

func (r *AgentRuns) Get(ctx context.Context, runID string) (*managementv1.GetAgentRunResponse, error) {
	client, err := r.client.requireSessionManagement()
	if err != nil {
		return nil, err
	}
	return client.GetAgentRun(ctx, &managementv1.GetAgentRunRequest{RunId: runID})
}
