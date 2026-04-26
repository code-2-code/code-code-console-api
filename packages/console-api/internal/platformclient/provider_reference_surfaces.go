package platformclient

import (
	"context"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func (t *Templates) List(ctx context.Context) ([]*managementv1.TemplateView, error) {
	client, err := t.client.requireProvider()
	if err != nil {
		return nil, err
	}
	response, err := client.ListTemplates(ctx, &providerservicev1.ListTemplatesRequest{})
	if err != nil {
		return nil, err
	}
	out := &managementv1.ListTemplatesResponse{}
	if err := transcodeProviderMessage(response, out); err != nil {
		return nil, err
	}
	return out.GetItems(), nil
}

func (t *Templates) Apply(ctx context.Context, templateID string, request *managementv1.ApplyTemplateRequest) (*managementv1.ApplyTemplateResult, error) {
	client, err := t.client.requireProvider()
	if err != nil {
		return nil, err
	}
	request.TemplateId = templateID
	providerRequest := &providerservicev1.ApplyTemplateRequest{}
	if err := transcodeProviderMessage(request, providerRequest); err != nil {
		return nil, err
	}
	response, err := client.ApplyTemplate(ctx, providerRequest)
	if err != nil {
		return nil, err
	}
	out := &managementv1.ApplyTemplateResponse{}
	if err := transcodeProviderMessage(response, out); err != nil {
		return nil, err
	}
	return out.GetResult(), nil
}

func (c *CLIDefinitions) List(ctx context.Context) ([]*managementv1.CLIDefinitionView, error) {
	client, err := c.client.requireProvider()
	if err != nil {
		return nil, err
	}
	response, err := client.ListCLIDefinitions(ctx, &providerservicev1.ListCLIDefinitionsRequest{})
	if err != nil {
		return nil, err
	}
	out := &managementv1.ListCLIDefinitionsResponse{}
	if err := transcodeProviderMessage(response, out); err != nil {
		return nil, err
	}
	return out.GetItems(), nil
}

func (s *SupportResources) ListVendors(ctx context.Context) ([]*supportv1.Vendor, error) {
	client, err := s.client.requireSupport()
	if err != nil {
		return nil, err
	}
	response, err := client.ListVendors(ctx, &supportv1.ListVendorsRequest{})
	if err != nil {
		return nil, err
	}
	return response.GetItems(), nil
}

func (s *SupportResources) ListCLIs(ctx context.Context) ([]*supportv1.CLI, error) {
	client, err := s.client.requireSupport()
	if err != nil {
		return nil, err
	}
	response, err := client.ListCLIs(ctx, &supportv1.ListCLIsRequest{})
	if err != nil {
		return nil, err
	}
	return response.GetItems(), nil
}

func (s *SupportResources) ListProviderSurfaces(ctx context.Context) ([]*providerv1.ProviderSurface, error) {
	client, err := s.client.requireSupport()
	if err != nil {
		return nil, err
	}
	response, err := client.ListProviderSurfaces(ctx, &supportv1.ListProviderSurfacesRequest{})
	if err != nil {
		return nil, err
	}
	return response.GetItems(), nil
}
