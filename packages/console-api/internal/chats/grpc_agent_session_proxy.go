package chats

import (
	"context"
	"io"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AgentSessionManagementProxy struct {
	managementv1.UnimplementedAgentSessionManagementServiceServer

	client managementv1.AgentSessionManagementServiceClient
}

func NewAgentSessionManagementProxy(client managementv1.AgentSessionManagementServiceClient) *AgentSessionManagementProxy {
	return &AgentSessionManagementProxy{client: client}
}

func (p *AgentSessionManagementProxy) GetAgentSession(ctx context.Context, request *managementv1.GetAgentSessionRequest) (*managementv1.GetAgentSessionResponse, error) {
	return p.client.GetAgentSession(ctx, request)
}

func (p *AgentSessionManagementProxy) CreateAgentSession(ctx context.Context, request *managementv1.CreateAgentSessionRequest) (*managementv1.CreateAgentSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "session setup is owned by the chat session repository")
}

func (p *AgentSessionManagementProxy) UpdateAgentSession(ctx context.Context, request *managementv1.UpdateAgentSessionRequest) (*managementv1.UpdateAgentSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "session setup is owned by the chat session repository")
}

func (p *AgentSessionManagementProxy) GetAgentSessionAction(ctx context.Context, request *managementv1.GetAgentSessionActionRequest) (*managementv1.GetAgentSessionActionResponse, error) {
	return p.client.GetAgentSessionAction(ctx, request)
}

func (p *AgentSessionManagementProxy) CreateAgentSessionAction(ctx context.Context, request *managementv1.CreateAgentSessionActionRequest) (*managementv1.CreateAgentSessionActionResponse, error) {
	return p.client.CreateAgentSessionAction(ctx, request)
}

func (p *AgentSessionManagementProxy) ResetAgentSessionWarmState(ctx context.Context, request *managementv1.ResetAgentSessionWarmStateRequest) (*managementv1.ResetAgentSessionWarmStateResponse, error) {
	return p.client.ResetAgentSessionWarmState(ctx, request)
}

func (p *AgentSessionManagementProxy) StopAgentSessionAction(ctx context.Context, request *managementv1.StopAgentSessionActionRequest) (*managementv1.StopAgentSessionActionResponse, error) {
	return p.client.StopAgentSessionAction(ctx, request)
}

func (p *AgentSessionManagementProxy) RetryAgentSessionAction(ctx context.Context, request *managementv1.RetryAgentSessionActionRequest) (*managementv1.RetryAgentSessionActionResponse, error) {
	return p.client.RetryAgentSessionAction(ctx, request)
}

func (p *AgentSessionManagementProxy) GetAgentRun(ctx context.Context, request *managementv1.GetAgentRunRequest) (*managementv1.GetAgentRunResponse, error) {
	return p.client.GetAgentRun(ctx, request)
}

func (p *AgentSessionManagementProxy) StreamAgentRunOutput(request *managementv1.StreamAgentRunOutputRequest, stream managementv1.AgentSessionManagementService_StreamAgentRunOutputServer) error {
	upstream, err := p.client.StreamAgentRunOutput(stream.Context(), request)
	if err != nil {
		return err
	}
	for {
		event, err := upstream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.Send(event); err != nil {
			return err
		}
	}
}
