package platformclient

import (
	"context"

	agentprofilev1 "code-code.internal/go-contract/platform/agent_profile/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

func (a *AgentProfiles) List(ctx context.Context) ([]*managementv1.AgentProfileListItem, error) {
	client, err := a.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.ListAgentProfiles(ctx, &managementv1.ListAgentProfilesRequest{})
	if err != nil {
		return nil, err
	}
	return response.GetItems(), nil
}

func (a *AgentProfiles) Get(ctx context.Context, profileID string) (*agentprofilev1.AgentProfile, error) {
	client, err := a.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.GetAgentProfile(ctx, &managementv1.GetAgentProfileRequest{ProfileId: profileID})
	if err != nil {
		return nil, err
	}
	return response.GetProfile(), nil
}

func (a *AgentProfiles) Create(ctx context.Context, request *managementv1.UpsertAgentProfileRequest) (*agentprofilev1.AgentProfile, error) {
	client, err := a.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.CreateAgentProfile(ctx, &managementv1.CreateAgentProfileRequest{Profile: request})
	if err != nil {
		return nil, err
	}
	return response.GetProfile(), nil
}

func (a *AgentProfiles) Update(ctx context.Context, profileID string, request *managementv1.UpsertAgentProfileRequest) (*agentprofilev1.AgentProfile, error) {
	client, err := a.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.UpdateAgentProfile(ctx, &managementv1.UpdateAgentProfileRequest{ProfileId: profileID, Profile: request})
	if err != nil {
		return nil, err
	}
	return response.GetProfile(), nil
}

func (a *AgentProfiles) Delete(ctx context.Context, profileID string) error {
	client, err := a.client.requireProfile()
	if err != nil {
		return err
	}
	_, err = client.DeleteAgentProfile(ctx, &managementv1.DeleteAgentProfileRequest{ProfileId: profileID})
	return err
}
