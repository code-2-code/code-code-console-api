package agentsessions

import (
	"context"

	agentcorev1 "code-code.internal/go-contract/agent/core/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

type ManagementClient struct {
	client managementv1.AgentSessionManagementServiceClient
}

func NewManagementClient(client managementv1.AgentSessionManagementServiceClient) *ManagementClient {
	return &ManagementClient{client: client}
}

func (c *ManagementClient) CreateTurn(ctx context.Context, sessionID string, actionID string, turnID string, runRequest *agentcorev1.RunRequest) (*managementv1.CreateAgentSessionActionResponse, error) {
	return c.client.CreateAgentSessionAction(ctx, &managementv1.CreateAgentSessionActionRequest{SessionId: sessionID, ActionId: actionID, TurnId: turnID, RunRequest: runRequest})
}

func (c *ManagementClient) ResetWarmState(ctx context.Context, sessionID string, actionID string) (*managementv1.ResetAgentSessionWarmStateResponse, error) {
	return c.client.ResetAgentSessionWarmState(ctx, &managementv1.ResetAgentSessionWarmStateRequest{SessionId: sessionID, ActionId: actionID})
}
