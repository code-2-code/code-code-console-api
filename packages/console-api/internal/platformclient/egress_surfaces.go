package platformclient

import (
	"context"

	egressv1 "code-code.internal/go-contract/egress/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

func (e *EgressPolicies) List(ctx context.Context) ([]*managementv1.EgressPolicyView, error) {
	client, err := e.client.requireEgress()
	if err != nil {
		return nil, err
	}
	response, err := client.ListEgressPolicies(ctx, &managementv1.ListEgressPoliciesRequest{})
	if err != nil {
		return nil, err
	}
	return response.GetItems(), nil
}

func (e *EgressPolicies) Update(ctx context.Context, policyID string, policy *egressv1.EgressPolicy) (*managementv1.EgressPolicyView, error) {
	client, err := e.client.requireEgress()
	if err != nil {
		return nil, err
	}
	response, err := client.UpdateEgressPolicy(ctx, &managementv1.UpdateEgressPolicyRequest{
		PolicyId: policyID,
		Policy:   policy,
	})
	if err != nil {
		return nil, err
	}
	return response.GetItem(), nil
}
