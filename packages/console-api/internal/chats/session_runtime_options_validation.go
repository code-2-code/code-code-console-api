package chats

import (
	"strconv"
	"strings"

	agentsessionv1 "code-code.internal/go-contract/platform/agent_session/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

func validateInlineSpecAgainstCatalog(spec *agentsessionv1.AgentSessionSpec, catalog *runtimeCatalog) error {
	if spec == nil {
		return grpcstatus.Error(codes.InvalidArgument, "inline spec is required")
	}
	providerID := strings.TrimSpace(spec.GetProviderId())
	provider, ok := catalog.providers[providerID]
	if !ok {
		return grpcstatus.Errorf(codes.InvalidArgument, "inline.providerId %q is not selectable", providerID)
	}
	executionClass := strings.TrimSpace(spec.GetExecutionClass())
	if _, ok := provider.executionClasses[executionClass]; !ok {
		return grpcstatus.Errorf(codes.InvalidArgument, "inline.executionClass %q is not selectable for providerId %q", executionClass, providerID)
	}
	runtimeConfig := spec.GetRuntimeConfig()
	if runtimeConfig == nil {
		return grpcstatus.Error(codes.InvalidArgument, "inline.runtimeConfig is required")
	}
	if err := validateRuntimeSurface(provider, "inline.runtimeConfig", runtimeConfig.GetProviderRuntimeRef(), primaryModelID(runtimeConfig)); err != nil {
		return err
	}
	for index, item := range runtimeConfig.GetFallbacks() {
		path := "inline.runtimeConfig.fallbacks[" + strconv.Itoa(index) + "]"
		if err := validateRuntimeSurface(provider, path, item.GetProviderRuntimeRef(), fallbackModelID(item)); err != nil {
			return err
		}
	}
	return nil
}

func validateRuntimeSurface(provider runtimeProviderCatalog, path string, runtimeRef *providerv1.ProviderRuntimeRef, modelID string) error {
	key := runtimeRefCatalogKey(runtimeRef)
	surface, ok := provider.surfaces[key]
	if !ok {
		return grpcstatus.Errorf(codes.InvalidArgument, "%s.providerRuntimeRef is not selectable", path)
	}
	if _, ok := surface.models[strings.TrimSpace(modelID)]; !ok {
		return grpcstatus.Errorf(codes.InvalidArgument, "%s.model %q is not selectable", path, strings.TrimSpace(modelID))
	}
	return nil
}

func primaryModelID(config *agentsessionv1.AgentSessionRuntimeConfig) string {
	if config == nil || config.GetPrimaryModelSelector() == nil {
		return ""
	}
	if modelID := strings.TrimSpace(config.GetPrimaryModelSelector().GetProviderModelId()); modelID != "" {
		return modelID
	}
	return strings.TrimSpace(config.GetPrimaryModelSelector().GetModelRef().GetModelId())
}

func fallbackModelID(item *agentsessionv1.AgentSessionRuntimeFallbackCandidate) string {
	if item == nil {
		return ""
	}
	if modelID := strings.TrimSpace(item.GetProviderModelId()); modelID != "" {
		return modelID
	}
	return strings.TrimSpace(item.GetModelRef().GetModelId())
}
