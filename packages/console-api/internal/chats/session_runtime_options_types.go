package chats

import providerv1 "code-code.internal/go-contract/provider/v1"

type sessionRuntimeOptionsView struct {
	Items []sessionRuntimeProviderOption `json:"items"`
}

type sessionRuntimeProviderOption struct {
	ProviderID       string                        `json:"providerId"`
	Label            string                        `json:"label"`
	ExecutionClasses []string                      `json:"executionClasses"`
	Surfaces         []sessionRuntimeSurfaceOption `json:"surfaces"`
}

type sessionRuntimeSurfaceOption struct {
	ProviderID string                       `json:"providerId"`
	Endpoint   *providerv1.ProviderEndpoint `json:"endpoint"`
	Label      string                       `json:"label"`
	Models     []string                     `json:"models"`
}

type runtimeCatalog struct {
	view      *sessionRuntimeOptionsView
	providers map[string]runtimeProviderCatalog
}

type runtimeProviderCatalog struct {
	executionClasses map[string]struct{}
	surfaces         map[string]runtimeSurfaceCatalog
}

type runtimeSurfaceCatalog struct {
	providerID string
	endpoint   *providerv1.ProviderEndpoint
	models     map[string]struct{}
}
