package chats

import (
	"sort"
	"strings"

	cliruntimev1 "code-code.internal/go-contract/platform/cli_runtime/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

func buildRuntimeCatalog(
	clis []*supportv1.CLI,
	availableImages []*cliruntimev1.CLIRuntimeImage,
	providerSurfaces []*managementv1.ProviderView,
) *runtimeCatalog {
	availableExecutionClasses := runtimeAvailableExecutionClasses(availableImages)

	sort.SliceStable(clis, func(i, j int) bool {
		return runtimeProviderLabel(clis[i]) < runtimeProviderLabel(clis[j])
	})

	items := make([]sessionRuntimeProviderOption, 0, len(clis))
	providers := make(map[string]runtimeProviderCatalog, len(clis))
	for _, cli := range clis {
		providerID := strings.TrimSpace(cli.GetCliId())
		if providerID == "" {
			continue
		}
		executionClasses := runtimeExecutionClasses(cli, availableExecutionClasses[providerID])
		surfaces, surfaceCatalog := runtimeProviderSurfaces(providerID, cli, providerSurfaces)
		if len(executionClasses) == 0 || len(surfaces) == 0 {
			continue
		}
		items = append(items, sessionRuntimeProviderOption{
			ProviderID:       providerID,
			Label:            runtimeProviderLabel(cli),
			ExecutionClasses: executionClasses,
			Surfaces:         surfaces,
		})
		providers[providerID] = runtimeProviderCatalog{
			executionClasses: setFromStrings(executionClasses),
			surfaces:         surfaceCatalog,
		}
	}
	return &runtimeCatalog{
		view:      &sessionRuntimeOptionsView{Items: items},
		providers: providers,
	}
}

func runtimeProviderSurfaces(
	providerID string,
	cli *supportv1.CLI,
	providerSurfaces []*managementv1.ProviderView,
) ([]sessionRuntimeSurfaceOption, map[string]runtimeSurfaceCatalog) {
	supportedProtocols := runtimeSupportedProtocols(cli)
	items := make([]sessionRuntimeSurfaceOption, 0, len(providerSurfaces))
	catalog := make(map[string]runtimeSurfaceCatalog, len(providerSurfaces))
	for _, surface := range providerSurfaces {
		accountID := strings.TrimSpace(surface.GetProviderId())
		surfaceID := strings.TrimSpace(surface.GetSurfaceId())
		models := runtimeSurfaceModels(surface)
		if accountID == "" || surfaceID == "" || len(models) == 0 {
			continue
		}
		for _, endpoint := range surface.GetEndpoints() {
			if !matchesRuntimeProvider(providerID, supportedProtocols, endpoint) {
				continue
			}
			key := runtimeEndpointCatalogKey(accountID, endpoint)
			if key == "" {
				continue
			}
			items = append(items, sessionRuntimeSurfaceOption{
				ProviderID: accountID,
				Endpoint:   proto.Clone(endpoint).(*providerv1.ProviderEndpoint),
				Label:      runtimeSurfaceLabel(surface, endpoint),
				Models:     models,
			})
			catalog[key] = runtimeSurfaceCatalog{
				providerID: accountID,
				endpoint:   proto.Clone(endpoint).(*providerv1.ProviderEndpoint),
				models:     setFromStrings(models),
			}
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Label == items[j].Label {
			return runtimeEndpointCatalogKey(items[i].ProviderID, items[i].Endpoint) < runtimeEndpointCatalogKey(items[j].ProviderID, items[j].Endpoint)
		}
		return items[i].Label < items[j].Label
	})
	return items, catalog
}

func runtimeAvailableExecutionClasses(images []*cliruntimev1.CLIRuntimeImage) map[string]map[string]struct{} {
	values := make(map[string]map[string]struct{})
	for _, image := range images {
		cliID := strings.TrimSpace(image.GetCliId())
		executionClass := strings.TrimSpace(image.GetExecutionClass())
		if cliID == "" || executionClass == "" || strings.TrimSpace(image.GetImage()) == "" {
			continue
		}
		if values[cliID] == nil {
			values[cliID] = make(map[string]struct{})
		}
		values[cliID][executionClass] = struct{}{}
	}
	return values
}

func runtimeExecutionClasses(cli *supportv1.CLI, available map[string]struct{}) []string {
	if cli == nil {
		return nil
	}
	if len(available) == 0 {
		return nil
	}
	values := make([]string, 0, len(cli.GetContainerImages()))
	seen := map[string]struct{}{}
	for _, item := range cli.GetContainerImages() {
		executionClass := strings.TrimSpace(item.GetExecutionClass())
		if executionClass == "" {
			continue
		}
		if _, ok := available[executionClass]; !ok {
			continue
		}
		if _, ok := seen[executionClass]; ok {
			continue
		}
		seen[executionClass] = struct{}{}
		values = append(values, executionClass)
	}
	return values
}

func runtimeSupportedProtocols(cli *supportv1.CLI) map[int32]struct{} {
	values := map[int32]struct{}{}
	for _, item := range cli.GetApiKeyProtocols() {
		values[int32(item.GetProtocol())] = struct{}{}
	}
	return values
}

func matchesRuntimeProvider(
	providerID string,
	supportedProtocols map[int32]struct{},
	endpoint *providerv1.ProviderEndpoint,
) bool {
	cliID := providerv1.EndpointCLIID(endpoint)
	if cliID != "" {
		return cliID == providerID
	}
	if len(supportedProtocols) == 0 {
		return false
	}
	_, ok := supportedProtocols[int32(providerv1.EndpointProtocol(endpoint))]
	return ok
}

func runtimeProviderLabel(item *supportv1.CLI) string {
	if label := strings.TrimSpace(item.GetDisplayName()); label != "" {
		return label
	}
	return strings.TrimSpace(item.GetCliId())
}

func runtimeSurfaceLabel(item *managementv1.ProviderView, endpoint *providerv1.ProviderEndpoint) string {
	if label := strings.TrimSpace(item.GetDisplayName()); label != "" {
		return label
	}
	if cliID := providerv1.EndpointCLIID(endpoint); cliID != "" {
		return cliID
	}
	if protocol := providerv1.EndpointProtocol(endpoint); protocol.String() != "" {
		return protocol.String()
	}
	return strings.TrimSpace(item.GetSurfaceId())
}

func runtimeSurfaceModels(surface *managementv1.ProviderView) []string {
	values := make([]string, 0, len(surface.GetModels()))
	seen := map[string]struct{}{}
	for _, item := range surface.GetModels() {
		modelID := strings.TrimSpace(item.GetProviderModelId())
		if modelID == "" {
			modelID = strings.TrimSpace(item.GetModelRef().GetModelId())
		}
		if modelID == "" {
			continue
		}
		if _, ok := seen[modelID]; ok {
			continue
		}
		seen[modelID] = struct{}{}
		values = append(values, modelID)
	}
	return values
}

func runtimeEndpointCatalogKey(providerID string, endpoint *providerv1.ProviderEndpoint) string {
	providerID = strings.TrimSpace(providerID)
	endpointKey := providerv1.EndpointKey(endpoint)
	if providerID == "" || endpointKey == "" {
		return ""
	}
	return providerID + "\x00" + endpointKey
}

func setFromStrings(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			result[trimmed] = struct{}{}
		}
	}
	return result
}
