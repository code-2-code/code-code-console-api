package providers

import providerv1 "code-code.internal/go-contract/provider/v1"

func testCLIProviderSurfaceRuntime(cliID string) *providerv1.ProviderSurfaceRuntime {
	return &providerv1.ProviderSurfaceRuntime{
		DisplayName: cliID,
		Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
		Access: &providerv1.ProviderSurfaceRuntime_Cli{
			Cli: &providerv1.ProviderCLISurfaceRuntime{CliId: cliID},
		},
	}
}

func testAPIProviderSurfaceRuntime() *providerv1.ProviderSurfaceRuntime {
	return &providerv1.ProviderSurfaceRuntime{
		DisplayName: "api",
		Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
		Access: &providerv1.ProviderSurfaceRuntime_Api{
			Api: &providerv1.ProviderAPISurfaceRuntime{},
		},
	}
}
