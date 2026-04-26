package platformclient

import (
	"context"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	mcpv1 "code-code.internal/go-contract/platform/mcp/v1"
)

func (m *MCPServers) List(ctx context.Context) ([]*managementv1.MCPServerListItem, error) {
	client, err := m.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.ListMCPServers(ctx, &managementv1.ListMCPServersRequest{})
	if err != nil {
		return nil, err
	}
	return response.GetItems(), nil
}

func (m *MCPServers) Get(ctx context.Context, mcpID string) (*mcpv1.MCPServer, error) {
	client, err := m.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.GetMCPServer(ctx, &managementv1.GetMCPServerRequest{McpId: mcpID})
	if err != nil {
		return nil, err
	}
	return response.GetMcp(), nil
}

func (m *MCPServers) Create(ctx context.Context, request *managementv1.UpsertMCPServerRequest) (*mcpv1.MCPServer, error) {
	client, err := m.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.CreateMCPServer(ctx, &managementv1.CreateMCPServerRequest{Mcp: request})
	if err != nil {
		return nil, err
	}
	return response.GetMcp(), nil
}

func (m *MCPServers) Update(ctx context.Context, mcpID string, request *managementv1.UpsertMCPServerRequest) (*mcpv1.MCPServer, error) {
	client, err := m.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.UpdateMCPServer(ctx, &managementv1.UpdateMCPServerRequest{McpId: mcpID, Mcp: request})
	if err != nil {
		return nil, err
	}
	return response.GetMcp(), nil
}

func (m *MCPServers) Delete(ctx context.Context, mcpID string) error {
	client, err := m.client.requireProfile()
	if err != nil {
		return err
	}
	_, err = client.DeleteMCPServer(ctx, &managementv1.DeleteMCPServerRequest{McpId: mcpID})
	return err
}
