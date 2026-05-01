package providers

import providerv1 "code-code.internal/go-contract/provider/v1"

func testCLIProviderEndpoint(cliID string) *providerv1.ProviderEndpoint {
	return &providerv1.ProviderEndpoint{
		Type: providerv1.ProviderEndpointType_PROVIDER_ENDPOINT_TYPE_CLI,
		Shape: &providerv1.ProviderEndpoint_Cli{Cli: &providerv1.ProviderCliEndpoint{
			CliId: cliID,
		}},
	}
}

func testAPIProviderEndpoint() *providerv1.ProviderEndpoint {
	return &providerv1.ProviderEndpoint{
		Type: providerv1.ProviderEndpointType_PROVIDER_ENDPOINT_TYPE_API,
		Shape: &providerv1.ProviderEndpoint_Api{Api: &providerv1.ProviderApiEndpoint{
			BaseUrl: "https://api.provider.test/v1",
		}},
	}
}
