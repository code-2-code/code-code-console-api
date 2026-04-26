package providers

import (
	"context"
	"fmt"
	"slices"
	"strings"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

type providerObservabilityProber interface {
	ProbeProvidersObservability(context.Context, []string) (*managementv1.ProbeProviderObservabilityResponse, error)
}

type providerProbeTarget struct {
	providerID string
}

func (s *ObservabilityService) ProbeAll(ctx context.Context) (*ProviderObservabilityProbeAllResponse, error) {
	if s == nil {
		return nil, fmt.Errorf("consoleapi/providers: observability service is nil")
	}
	if s.prober == nil {
		return nil, fmt.Errorf("consoleapi/providers: observability prober is nil")
	}
	providers, err := s.providers.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	targets := sortedProviderProbeTargets(providers)
	if len(targets) == 0 {
		return &ProviderObservabilityProbeAllResponse{Message: "no providers to probe"}, nil
	}
	response, err := s.prober.ProbeProvidersObservability(ctx, providerProbeTargetIDs(targets))
	if err != nil {
		return nil, err
	}
	return &ProviderObservabilityProbeAllResponse{
		TriggeredCount: len(targets),
		Message:        strings.TrimSpace(response.GetMessage()),
		Results: []ProviderObservabilityProbeState{{
			Outcome: managementv1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_UNSPECIFIED.String(),
			Message: strings.TrimSpace(response.GetMessage()),
		}},
	}, nil
}

func (s *ObservabilityService) ProbeProviders(ctx context.Context, providerIDs []string) (*ProviderObservabilityProbeAllResponse, error) {
	if s == nil {
		return nil, fmt.Errorf("consoleapi/providers: observability service is nil")
	}
	if s.prober == nil {
		return nil, fmt.Errorf("consoleapi/providers: observability prober is nil")
	}
	ids := normalizedProviderProbeIDs(providerIDs)
	if len(ids) == 0 {
		return &ProviderObservabilityProbeAllResponse{Message: "no providers to probe"}, nil
	}
	response, err := s.prober.ProbeProvidersObservability(ctx, ids)
	if err != nil {
		return nil, err
	}
	return &ProviderObservabilityProbeAllResponse{
		TriggeredCount: len(ids),
		WorkflowID:     strings.TrimSpace(response.GetWorkflowId()),
		Message:        strings.TrimSpace(response.GetMessage()),
		Results: []ProviderObservabilityProbeState{{
			Outcome: managementv1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_UNSPECIFIED.String(),
			Message: strings.TrimSpace(response.GetMessage()),
		}},
	}, nil
}

func providerProbeTargetIDs(targets []providerProbeTarget) []string {
	out := make([]string, 0, len(targets))
	for _, target := range targets {
		if target.providerID != "" {
			out = append(out, target.providerID)
		}
	}
	return out
}

func normalizedProviderProbeIDs(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func sortedProviderProbeTargets(providers []*managementv1.ProviderView) []providerProbeTarget {
	targetByKey := map[string]providerProbeTarget{}
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		providerID := strings.TrimSpace(provider.GetProviderId())
		if providerID == "" || len(provider.GetSurfaces()) == 0 {
			continue
		}
		for _, surface := range provider.GetSurfaces() {
			owner := providerSurfaceBindingOwner(surface)
			if owner.kind == "" || owner.id == "" {
				continue
			}
			target := providerProbeTarget{providerID: providerID}
			key := providerProbeTargetKey(target)
			if current, ok := targetByKey[key]; ok {
				targetByKey[key] = mergeProviderProbeTarget(current, target)
				continue
			}
			targetByKey[key] = target
		}
	}
	targets := make([]providerProbeTarget, 0, len(targetByKey))
	for _, target := range targetByKey {
		targets = append(targets, target)
	}
	slices.SortFunc(targets, func(left, right providerProbeTarget) int {
		return strings.Compare(left.providerID, right.providerID)
	})
	return targets
}

func providerProbeTargetKey(target providerProbeTarget) string {
	return "provider:" + strings.TrimSpace(target.providerID)
}

func mergeProviderProbeTarget(current, candidate providerProbeTarget) providerProbeTarget {
	if strings.TrimSpace(current.providerID) == "" {
		current.providerID = strings.TrimSpace(candidate.providerID)
	}
	return current
}
