package chats

import (
	"sort"
	"strings"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	cliruntimev1 "code-code.internal/go-contract/platform/cli_runtime/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

func buildRuntimeCatalog(
	clis []*supportv1.CLI,
	cliDefinitions []*managementv1.CLIDefinitionView,
	availableImages []*cliruntimev1.CLIRuntimeImage,
	providerSurfaces []*managementv1.ProviderSurfaceBindingView,
) *runtimeCatalog {
	definitionByID := make(map[string]*managementv1.CLIDefinitionView, len(cliDefinitions))
	for _, item := range cliDefinitions {
		cliID := strings.TrimSpace(item.GetCliId())
		if cliID != "" {
			definitionByID[cliID] = item
		}
	}
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
		executionClasses := runtimeExecutionClasses(definitionByID[providerID], availableExecutionClasses[providerID])
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
	providerSurfaces []*managementv1.ProviderSurfaceBindingView,
) ([]sessionRuntimeSurfaceOption, map[string]runtimeSurfaceCatalog) {
	supportedProtocols := runtimeSupportedProtocols(cli)
	items := make([]sessionRuntimeSurfaceOption, 0, len(providerSurfaces))
	catalog := make(map[string]runtimeSurfaceCatalog, len(providerSurfaces))
	for _, surface := range providerSurfaces {
		if !matchesRuntimeProvider(providerID, supportedProtocols, surface) {
			continue
		}
		surfaceID := strings.TrimSpace(surface.GetSurfaceId())
		models := runtimeSurfaceModels(surface)
		if surfaceID == "" || len(models) == 0 {
			continue
		}
		runtimeRef := runtimeRefForSurface(surface)
		if runtimeRef == nil {
			continue
		}
		key := runtimeRefCatalogKey(runtimeRef)
		items = append(items, sessionRuntimeSurfaceOption{
			RuntimeRef: runtimeRef,
			Label:      runtimeSurfaceLabel(surface),
			Models:     models,
		})
		catalog[key] = runtimeSurfaceCatalog{
			runtimeRef: proto.Clone(runtimeRef).(*providerv1.ProviderRuntimeRef),
			models:     setFromStrings(models),
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Label == items[j].Label {
			return runtimeRefCatalogKey(items[i].RuntimeRef) < runtimeRefCatalogKey(items[j].RuntimeRef)
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

func runtimeExecutionClasses(definition *managementv1.CLIDefinitionView, available map[string]struct{}) []string {
	if definition == nil {
		return nil
	}
	if len(available) == 0 {
		return nil
	}
	values := make([]string, 0, len(definition.GetContainerImages()))
	seen := map[string]struct{}{}
	for _, item := range definition.GetContainerImages() {
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
	surface *managementv1.ProviderSurfaceBindingView,
) bool {
	runtimeOwnerID := providerv1.RuntimeCLIID(surface.GetRuntime())
	if runtimeOwnerID != "" {
		return runtimeOwnerID == providerID
	}
	if len(supportedProtocols) == 0 {
		return false
	}
	_, ok := supportedProtocols[int32(providerv1.RuntimeProtocol(surface.GetRuntime()))]
	return ok
}

func runtimeProviderLabel(item *supportv1.CLI) string {
	if label := strings.TrimSpace(item.GetDisplayName()); label != "" {
		return label
	}
	return strings.TrimSpace(item.GetCliId())
}

func runtimeSurfaceLabel(item *managementv1.ProviderSurfaceBindingView) string {
	if label := strings.TrimSpace(item.GetProviderDisplayName()); label != "" {
		return label
	}
	if label := strings.TrimSpace(item.GetRuntime().GetDisplayName()); label != "" {
		return label
	}
	if label := strings.TrimSpace(item.GetDisplayName()); label != "" {
		return label
	}
	return strings.TrimSpace(item.GetSurfaceId())
}

func runtimeSurfaceModels(surface *managementv1.ProviderSurfaceBindingView) []string {
	values := make([]string, 0, len(surface.GetRuntime().GetCatalog().GetModels())+1)
	seen := map[string]struct{}{}
	for _, item := range surface.GetRuntime().GetCatalog().GetModels() {
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

func runtimeRefForSurface(surface *managementv1.ProviderSurfaceBindingView) *providerv1.ProviderRuntimeRef {
	if surface == nil || surface.GetRuntime() == nil {
		return nil
	}
	ref := &providerv1.ProviderRuntimeRef{
		ProviderId: strings.TrimSpace(surface.GetProviderId()),
		SurfaceId:  strings.TrimSpace(surface.GetSurfaceId()),
	}
	switch access := surface.GetRuntime().GetAccess().(type) {
	case *providerv1.ProviderSurfaceRuntime_Api:
		ref.Access = &providerv1.ProviderRuntimeRef_Api{
			Api: &providerv1.ProviderRuntimeAPIRef{Protocol: access.Api.GetProtocol()},
		}
	case *providerv1.ProviderSurfaceRuntime_Cli:
		ref.Access = &providerv1.ProviderRuntimeRef_Cli{Cli: &providerv1.ProviderRuntimeCLIRef{}}
	default:
		return nil
	}
	if ref.GetProviderId() == "" || ref.GetSurfaceId() == "" {
		return nil
	}
	return ref
}

func runtimeRefCatalogKey(ref *providerv1.ProviderRuntimeRef) string {
	if ref == nil {
		return ""
	}
	parts := []string{
		strings.TrimSpace(ref.GetProviderId()),
		strings.TrimSpace(ref.GetSurfaceId()),
	}
	switch access := ref.GetAccess().(type) {
	case *providerv1.ProviderRuntimeRef_Api:
		parts = append(parts, "api", protocolKey(access.Api.GetProtocol()))
	case *providerv1.ProviderRuntimeRef_Cli:
		parts = append(parts, "cli")
	default:
		parts = append(parts, "unspecified")
	}
	return strings.Join(parts, "\x00")
}

func protocolKey(protocol apiprotocolv1.Protocol) string {
	return strings.TrimSpace(protocol.String())
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
